package handler

import (
	"fmt"
	"real-estate-backend/pkg/upload"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type UploadHandler struct {
	uploadService upload.UploadService
}

func NewUploadHandler(uploadService upload.UploadService) *UploadHandler {
	return &UploadHandler{uploadService}
}

func (h *UploadHandler) UploadImage(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		fmt.Printf("Upload Error (Image): %v\n", err)
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "No file provided: "+err.Error())
	}

	f, err := file.Open()
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to open file")
	}
	defer f.Close()

	url, err := h.uploadService.UploadImage(f, "properties")
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Upload failed: "+err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Image uploaded successfully", fiber.Map{
		"url": url,
	})
}

func (h *UploadHandler) UploadDocument(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		fmt.Printf("Upload Error (Doc): %v\n", err)
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "No document provided: "+err.Error())
	}

	f, err := file.Open()
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to open document")
	}
	defer f.Close()

	url, err := h.uploadService.UploadDocument(f, file.Filename, "documents")
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Document upload failed: "+err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Document uploaded successfully", fiber.Map{
		"url": url,
	})
}
