package handler

import (
	"log"
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"
	websocketpkg "real-estate-backend/pkg/websocket"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type MessageHandler struct {
	service   service.MessageService
	hub       *websocketpkg.Hub
	jwtSecret string
}

func NewMessageHandler(service service.MessageService, hub *websocketpkg.Hub, jwtSecret string) *MessageHandler {
	return &MessageHandler{service, hub, jwtSecret}
}

func (h *MessageHandler) HandleWS(c *websocket.Conn) {
	tokenString := c.Query("token")
	if tokenString == "" {
		log.Printf("WS Error: No token provided")
		c.Close()
		return
	}

	// Verify token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		log.Printf("WS Error: Invalid token")
		c.Close()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("WS Error: Invalid claims")
		c.Close()
		return
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok || userIDStr == "" {
		log.Printf("WS Error: Missing user_id claim")
		c.Close()
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("WS Error: Invalid UserID: %v", err)
		c.Close()
		return
	}

	log.Printf("User %s attempting to connect...", userID)
	websocketpkg.ServeWs(h.hub, c.Conn, userID)
}

func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {
	senderID := c.Locals("user_id").(string)
	var req struct {
		ReceiverID string `json:"receiver_id"`
		Content    string `json:"content"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	msg, err := h.service.SendMessage(senderID, req.ReceiverID, req.Content)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to send message")
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, "Message sent", msg)
}

func (h *MessageHandler) GetConversations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	conversations, err := h.service.GetConversations(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch conversations")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Conversations retrieved", conversations)
}

func (h *MessageHandler) GetChatHistory(c *fiber.Ctx) error {
	user1ID := c.Locals("user_id").(string)
	user2ID := c.Params("id")

	history, err := h.service.GetChatHistory(user1ID, user2ID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch chat history")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "History retrieved", history)
}
