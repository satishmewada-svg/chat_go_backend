package services

import (
	"my-ecomm/config"
	"my-ecomm/models"
	"sync"
	"time"
)

type PresenceService struct {
	mu              sync.RWMutex
	onlineUsers     map[uint]time.Time // userID -> last heartbeat time
	heartbeatTicker *time.Ticker
}

var presenceInstance *PresenceService
var presenceOnce sync.Once

func GetPresenceService() *PresenceService {
	presenceOnce.Do(func() {
		presenceInstance = &PresenceService{
			onlineUsers:     make(map[uint]time.Time),
			heartbeatTicker: time.NewTicker(30 * time.Second),
		}
		go presenceInstance.checkOfflineUsers()
	})
	return presenceInstance
}

// UserConnected marks a user as online
func (ps *PresenceService) UserConnected(userID uint) error {
	ps.mu.Lock()
	ps.onlineUsers[userID] = time.Now()
	ps.mu.Unlock()

	// Update database
	return config.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"is_online":    true,
			"last_seen_at": time.Now(),
		}).Error
}

// UserDisconnected marks a user as offline
func (ps *PresenceService) UserDisconnected(userID uint) error {
	ps.mu.Lock()
	delete(ps.onlineUsers, userID)
	ps.mu.Unlock()

	now := time.Now()
	return config.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"is_online":    false,
			"last_seen_at": now,
		}).Error
}

// Heartbeat updates user's last active time
func (ps *PresenceService) Heartbeat(userID uint) {
	ps.mu.Lock()
	ps.onlineUsers[userID] = time.Now()
	ps.mu.Unlock()

	// Update last seen
	config.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_seen_at", time.Now())
}

// IsUserOnline checks if a user is currently online
func (ps *PresenceService) IsUserOnline(userID uint) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	lastSeen, exists := ps.onlineUsers[userID]
	if !exists {
		return false
	}

	// Consider offline if no heartbeat in last 2 minutes
	return time.Since(lastSeen) < 2*time.Minute
}

// GetOnlineUsers returns list of online user IDs
func (ps *PresenceService) GetOnlineUsers() []uint {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var onlineUserIDs []uint
	now := time.Now()

	for userID, lastSeen := range ps.onlineUsers {
		if now.Sub(lastSeen) < 2*time.Minute {
			onlineUserIDs = append(onlineUserIDs, userID)
		}
	}

	return onlineUserIDs
}

// checkOfflineUsers periodically checks for users who went offline
func (ps *PresenceService) checkOfflineUsers() {
	for range ps.heartbeatTicker.C {
		ps.mu.Lock()
		now := time.Now()

		for userID, lastSeen := range ps.onlineUsers {
			// If no heartbeat in 2 minutes, mark as offline
			if now.Sub(lastSeen) > 2*time.Minute {
				delete(ps.onlineUsers, userID)

				// Update database
				go func(uid uint) {
					config.DB.Model(&models.User{}).
						Where("id = ?", uid).
						Updates(map[string]interface{}{
							"is_online":    false,
							"last_seen_at": time.Now(),
						})
				}(userID)
			}
		}

		ps.mu.Unlock()
	}
}
