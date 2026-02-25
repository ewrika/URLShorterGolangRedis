package routes

import (
	"UrlShorter/database"
	"UrlShorter/helpers"
	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"os"
	"strconv"
	"time"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := new(request)

	if err := c.BodyParser(body); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "you cant hack the system",
		})
	}

	// ----- RATE LIMIT (db 1) -----
	rLimit := database.CreateClient(1)
	defer rLimit.Close()

	ip := c.IP()
	val, err := rLimit.Get(database.Ctx, ip).Result()
	if err == redis.Nil {
		_ = rLimit.Set(database.Ctx, ip, os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else if err == nil {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := rLimit.TTL(database.Ctx, ip).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":      "Rate Limit exceeded",
				"rate_limit": limit,
			})
		}
	} else {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to connect rate limiter",
		})
	}

	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid URL"})
	}

	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Invalid URL"})
	}

	body.URL = helpers.EnforceHTTP(body.URL)

	if body.Expiry == 0 {
		body.Expiry = 24
	}
	expiry := body.Expiry * 3600 * time.Second

	rStore := database.CreateClient(0)
	defer rStore.Close()

	urlKey := "url:" + body.URL
	if existingID, err := rStore.Get(database.Ctx, urlKey).Result(); err == nil && existingID != "" {
		rLimit.Decr(database.Ctx, ip)
		remainingStr, _ := rLimit.Get(database.Ctx, ip).Result()
		remaining, _ := strconv.Atoi(remainingStr)
		ttl, _ := rLimit.TTL(database.Ctx, ip).Result()

		resp := response{
			URL:             body.URL,
			CustomShort:     os.Getenv("DOMAIN") + "/" + existingID,
			Expiry:          body.Expiry,
			XRateRemaining:  remaining,
			XRateLimitReset: ttl / time.Nanosecond / time.Minute,
		}
		return c.Status(fiber.StatusOK).JSON(resp)
	}

	var id string
	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	if val, _ := rStore.Get(database.Ctx, id).Result(); val != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You cannot shorten this URL",
		})
	}

	if err := rStore.Set(database.Ctx, id, body.URL, expiry).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to connect server",
		})
	}
	if err := rStore.Set(database.Ctx, urlKey, id, expiry).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to connect server",
		})
	}

	rLimit.Decr(database.Ctx, ip)
	remainingStr, _ := rLimit.Get(database.Ctx, ip).Result()
	remaining, _ := strconv.Atoi(remainingStr)
	ttl, _ := rLimit.TTL(database.Ctx, ip).Result()

	resp := response{
		URL:             body.URL,
		CustomShort:     os.Getenv("DOMAIN") + "/" + id,
		Expiry:          body.Expiry,
		XRateRemaining:  remaining,
		XRateLimitReset: ttl / time.Nanosecond / time.Minute,
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
