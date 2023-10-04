package cron

import (
	"MedKick-backend/pkg/database/models"
	"MedKick-backend/pkg/echo/dto"
	mioApi "MedKick-backend/pkg/mio/api"
	"MedKick-backend/pkg/validator"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"time"
)

// syncDevices godoc
// @Summary Sync Devices from Mio Connect
// @Description CRON ONLY - Pulls and Syncs devices from Mio-Connect
// @Tags CRON
// @Accept json
// @Produce json
// @Success 200
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /cron/sync-devices [get]
func syncDevices(c echo.Context) error {
	var req Request
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Failed to bind request",
		})
	}

	if err := validator.Validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid request",
		})
	}

	if req.Token != os.Getenv("CRON_SECRET") {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Invalid token",
		})
	}

	mioClient := mioApi.NewClient(os.Getenv("MIO_API_KEY"))
	mioDevices, err := mioClient.GetDeviceList()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to get devices from Mio Connect",
		})
	}

	for _, mioDevice := range mioDevices.Items {
		device := models.Device{
			IMEI: mioDevice.IMEI,
		}
		if err := device.GetDeviceByIMEI(); err != nil {
			// Create new device
			fetchDevice, err := mioClient.GetDevice(mioDevice.DeviceID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
					Error: "Failed to get device from Mio Connect",
				})
			}

			device.ModelNumber = fetchDevice.ModelNumber
			device.DeviceID = fetchDevice.DeviceID
			device.SerialNumber = fetchDevice.SerialNumber
			device.CreatedAt = fetchDevice.CreatedAt
			device.FirmwareVersion = fetchDevice.FirmwareVersion
			device.UpdatedAt = time.Now()
			device.Name = ""
			device.BatteryLevel = 0
			device.SignalStrength = ""
			device.UserID = 0

			if err := device.CreateDevice(); err != nil {
				return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
					Error: "Failed to create device",
				})
			}
		}

		// Update device
		fetchDevice, err := mioClient.GetDevice(mioDevice.DeviceID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Error: "Failed to get device from Mio Connect",
			})
		}

		device.FirmwareVersion = fetchDevice.FirmwareVersion
		device.UpdatedAt = time.Now()
		if err := device.UpdateDevice(); err != nil {
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Error: "Failed to update device",
			})
		}
	}

	return c.NoContent(http.StatusOK)
}