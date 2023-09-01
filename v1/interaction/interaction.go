package interaction

import (
	"MedKick-backend/pkg/database/models"
	"MedKick-backend/pkg/echo/dto"
	"MedKick-backend/pkg/echo/middleware"
	"MedKick-backend/pkg/validator"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
	"time"
)

type CreateRequest struct {
	UserID      uint   `json:"user_id" validate:"required"`
	DoctorID    *uint  `json:"doctor_id"`
	Duration    uint   `json:"duration" validate:"required"`
	Notes       string `json:"notes" validate:"required"`
	SessionDate string `json:"session_date" validate:"required" example:"2021-01-01T00:00:00Z"`
}

func createInteraction(c echo.Context) error {
	var request CreateRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
		})
	}

	if err := validator.Validate.Struct(request); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
		})
	}

	var sessionDate time.Time
	if err := sessionDate.UnmarshalText([]byte(request.SessionDate)); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "The session date must be in the RFC 3339 format",
		})
	}

	self := middleware.GetSelf(c)
	if self.Role == "doctor" {
		request.DoctorID = self.ID
	}

	if self.Role == "admin" {
		if request.DoctorID == nil {
			return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error: "Admins must specify a doctor ID",
			})
		}
		// Check if doctor exists
		doctor := models.User{
			ID: request.DoctorID,
		}
		if err := doctor.GetUser(); err != nil {
			return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error: "Doctor does not exist",
			})
		}
	}

	// Check if user (patient) exists
	user := models.User{
		ID: &request.UserID,
	}
	if err := user.GetUser(); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "User does not exist",
		})
	}

	i := models.Interaction{
		UserID:      request.UserID,
		DoctorID:    *request.DoctorID,
		Duration:    request.Duration,
		Notes:       request.Notes,
		SessionDate: sessionDate,
	}
	if err := i.CreateInteraction(); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to create interaction",
		})
	}

	return c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Successfully created interaction",
	})
}

func getInteraction(c echo.Context) error {
	id := c.Param("id")

	self := middleware.GetSelf(c)
	if id == "" {
		if self.Role == "admin" {
			interactions, err := models.GetInteractions()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
					Error: "Failed to get interactions",
				})
			}
			return c.JSON(http.StatusOK, interactions)
		} else if self.Role == "doctor" {
			interactions, err := models.GetInteractionsByDoctor(*self.ID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
					Error: "Failed to get interactions",
				})
			}
			return c.JSON(http.StatusOK, interactions)
		} else {
			interactions, err := models.GetInteractionsByUser(*self.ID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
					Error: "Failed to get interactions",
				})
			}
			return c.JSON(http.StatusOK, interactions)
		}
	}

	idInt, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid ID",
		})
	}
	i := models.Interaction{
		ID: uint(idInt),
	}
	if err := i.GetInteraction(); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to get interaction",
		})
	}

	if self.Role == "admin" || (self.Role == "doctor" && self.ID == &i.DoctorID) || (self.Role == "patient" && self.ID == &i.UserID) {
		return c.JSON(http.StatusOK, i)
	}

	return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
		Error: "Unauthorized",
	})
}

type UpdateRequest struct {
	UserID      *uint  `json:"user_id"`
	DoctorID    *uint  `json:"doctor_id"`
	Duration    *uint  `json:"duration"`
	Notes       string `json:"notes"`
	SessionDate string `json:"session_date" example:"2021-01-01T00:00:00Z"`
}

func updateInteraction(c echo.Context) error {
	var request UpdateRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
		})
	}

	if err := validator.Validate.Struct(request); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
		})
	}

	id := c.Param("id")
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid ID",
		})
	}

	i := models.Interaction{
		ID: uint(idUint),
	}
	if err := i.GetInteraction(); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to get interaction",
		})
	}

	self := middleware.GetSelf(c)
	if self.Role == "admin" || (self.Role == "doctor" && self.ID == &i.DoctorID) {
		if request.UserID != nil {
			i.UserID = *request.UserID
		}
		if request.DoctorID != nil {
			i.DoctorID = *request.DoctorID
		}
		if request.Duration != nil {
			i.Duration = *request.Duration
		}
		if request.Notes != "" {
			i.Notes = request.Notes
		}
		if request.SessionDate != "" {
			var sessionDate time.Time
			if err := sessionDate.UnmarshalText([]byte(request.SessionDate)); err != nil {
				return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
					Error: "The session date must be in the RFC 3339 format",
				})
			}
			i.SessionDate = sessionDate
		}

		if err := i.UpdateInteraction(); err != nil {
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Error: "Failed to update interaction",
			})
		}
		return c.JSON(http.StatusOK, i)
	}

	return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
		Error: "You may only update your own interactions",
	})
}

func deleteInteraction(c echo.Context) error {
	id := c.Param("id")
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid ID",
		})
	}

	i := models.Interaction{
		ID: uint(idUint),
	}
	if err := i.GetInteraction(); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to get interaction",
		})
	}

	self := middleware.GetSelf(c)
	if self.Role == "admin" || (self.Role == "doctor" && self.ID == &i.DoctorID) {
		if err := i.DeleteInteraction(); err != nil {
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Error: "Failed to delete interaction",
			})
		}
		return c.JSON(http.StatusOK, dto.MessageResponse{
			Message: "Successfully deleted interaction",
		})
	}

	return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
		Error: "You may only delete your own interactions",
	})
}
