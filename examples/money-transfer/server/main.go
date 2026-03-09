/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/zigflow/graphs/contributors>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

type Transfer struct {
	Amount         float32 `json:"amount" validate:"required,gte=0"`
	Attempt        int32   `json:"attempt" validate:"required,gt=0"`
	IdempotencyKey string  `json:"idempotencyKey" validate:"required,uuid4"`
	Name           string  `json:"name" validate:"required"`
}

type TransferInput struct {
	Amount      int    `json:"amount" validate:"required,gte=0"`
	FromAccount string `json:"fromAccount" validate:"required"`
	ToAccount   string `json:"toAccount" validate:"required"`
}

const (
	API_DOWNTIME    = "AccountTransferWorkflowAPIDowntime"
	INVALID_ACCOUNT = "AccountTransferWorkflowInvalidAccount"
)

func getTransfer(c *fiber.Ctx, validate *validator.Validate) (*Transfer, error) {
	var transfer Transfer

	if err := c.BodyParser(&transfer); err != nil {
		fmt.Println(err)
		return nil, err
	}

	if err := validate.Struct(transfer); err != nil {
		fmt.Println(err)
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return &transfer, nil
}

func main() {
	app := fiber.New()
	validate := validator.New()

	app.
		Use(logger.New()).
		Use(func(c *fiber.Ctx) error {
			// Log everything received - terrible for security, but ok for this demo
			fmt.Println(string(c.Body()))
			return c.Next()
		})

	app.Post("/deposit", func(c *fiber.Ctx) error {
		transfer, err := getTransfer(c, validate)
		if err != nil {
			return err
		}

		// simulate external API call
		error := simulateExternalOperationWithError(1000, transfer.Name, transfer.Attempt)
		if INVALID_ACCOUNT == error {
			// a business error, which cannot be retried
			return fiber.NewError(fiber.StatusBadRequest, "deposit activity failed, account is invalid")
		}

		return c.SendString("SUCCESS")
	})

	app.Post("/notify", func(c *fiber.Ctx) error {
		var input TransferInput

		if err := c.BodyParser(&input); err != nil {
			fmt.Println(err)
			return err
		}

		if err := validate.Struct(input); err != nil {
			fmt.Println(err)
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		fmt.Printf("%+v\n", input)

		simulateExternalOperation(1000)

		return c.SendString("SUCCESS")
	})

	app.Post("/validate", func(c *fiber.Ctx) error {
		simulateExternalOperation(1000)

		return c.SendString("SUCCESS")
	})

	app.Post("/withdraw", func(c *fiber.Ctx) error {
		transfer, err := getTransfer(c, validate)
		if err != nil {
			return err
		}

		// simulate external API call
		error := simulateExternalOperationWithError(1000, transfer.Name, transfer.Attempt)
		if API_DOWNTIME == error {
			return fiber.ErrServiceUnavailable
		}

		return c.SendString("SUCCESS")
	})

	app.Listen(":3000")
}

// @link https://github.com/temporal-sa/money-transfer-demo/blob/d6415f56730f041a9c219b064638690b74e2643f/go/activities/shared.go#L5
func simulateExternalOperation(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// @link https://github.com/temporal-sa/money-transfer-demo/blob/d6415f56730f041a9c219b064638690b74e2643f/go/activities/shared.go#L9
func simulateExternalOperationWithError(ms int, name string, attempt int32) string {
	simulateExternalOperation(ms / int(attempt))
	var result string
	if attempt < 5 {
		result = name
	} else {
		result = "NoError"
	}
	return result
}
