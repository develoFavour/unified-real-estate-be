package service

import (
	"encoding/json"
	"log"
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"
	"real-estate-backend/pkg/websocket"

	"github.com/google/uuid"
)

type MessageService interface {
	SendMessage(senderID, receiverID, content string) (*models.Message, error)
	GetConversations(userID string) (any, error)
	GetChatHistory(user1ID, user2ID string) ([]models.Message, error)
}

type messageService struct {
	repo repository.MessageRepository
	hub  *websocket.Hub
}

func NewMessageService(repo repository.MessageRepository, hub *websocket.Hub) MessageService {
	return &messageService{repo, hub}
}

func (s *messageService) SendMessage(senderID, receiverID, content string) (*models.Message, error) {
	sID, _ := uuid.Parse(senderID)
	rID, _ := uuid.Parse(receiverID)

	msg := &models.Message{
		SenderID:   sID,
		ReceiverID: rID,
		Content:    content,
	}

	if err := s.repo.SaveMessage(msg); err != nil {
		return nil, err
	}

	// Broadcast via WebSocket to both receiver and sender (for multi-tab sync)
	payload, _ := json.Marshal(msg)
	log.Printf("Broadcasting message from %s to %s", msg.SenderID, msg.ReceiverID)
	s.hub.BroadcastToUser(msg.ReceiverID, payload)
	s.hub.BroadcastToUser(msg.SenderID, payload)

	return msg, nil
}

func (s *messageService) GetConversations(userID string) (any, error) {
	return s.repo.GetConversations(userID)
}

func (s *messageService) GetChatHistory(user1ID, user2ID string) ([]models.Message, error) {
	return s.repo.GetChatHistory(user1ID, user2ID)
}
