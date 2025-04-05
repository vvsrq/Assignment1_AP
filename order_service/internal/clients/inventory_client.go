package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type InventoryProductResponse struct {
	Status  string  `json:"Status"`
	Message string  `json:"Message"`
	Data    Product `json:"Data"`
}

type InventoryClient interface {
	GetProduct(productID int) (*Product, error)
	UpdateStock(productID int, newStock int) error
}

type inventoryHTTPClient struct {
	baseURL string
	client  *http.Client
	log     *logrus.Logger
}

func NewInventoryHTTPClient(baseURL string, timeout time.Duration, logger *logrus.Logger) InventoryClient {
	return &inventoryHTTPClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		log: logger,
	}
}

func (c *inventoryHTTPClient) GetProduct(productID int) (*Product, error) {
	url := fmt.Sprintf("%s/products/%d", c.baseURL, productID)
	c.log.Infof("InventoryClient: Requesting product info from URL: %s", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		c.log.Errorf("InventoryClient: Failed to create GetProduct request for ID %d: %v", productID, err)
		return nil, fmt.Errorf("failed to create inventory request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Errorf("InventoryClient: Failed to execute GetProduct request for ID %d: %v", productID, err)
		return nil, fmt.Errorf("failed to communicate with inventory service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.log.Warnf("InventoryClient: Product with ID %d not found (status %d)", productID, resp.StatusCode)
		return nil, fmt.Errorf("product with ID %d not found in inventory", productID)
	}

	if resp.StatusCode != http.StatusOK {
		c.log.Errorf("InventoryClient: GetProduct request for ID %d failed with status %d", productID, resp.StatusCode)

		return nil, fmt.Errorf("inventory service returned status %d for product %d", resp.StatusCode, productID)
	}

	var response InventoryProductResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.log.Errorf("InventoryClient: Failed to decode GetProduct response for ID %d: %v", productID, err)
		return nil, fmt.Errorf("failed to decode inventory response: %w", err)
	}

	c.log.Infof("InventoryClient: Parsed product data for ID %d: ID=%d, Name='%s', Stock=%d, Price=%.2f",
		productID, response.Data.ID, response.Data.Name, response.Data.Stock, response.Data.Price)

	if response.Data.ID != productID {
		c.log.Warnf("InventoryClient: Mismatched product ID in response. Requested %d, got %d", productID, response.Data.ID)
	}

	return &response.Data, nil
}

func (c *inventoryHTTPClient) UpdateStock(productID int, newStock int) error {
	if newStock < 0 {
		c.log.Errorf("InventoryClient: Attempted to set negative stock (%d) for product ID %d", newStock, productID)
		return fmt.Errorf("stock cannot be negative")
	}

	url := fmt.Sprintf("%s/products/%d", c.baseURL, productID)

	updateData := map[string]int{"stock": newStock}
	jsonData, err := json.Marshal(updateData)
	if err != nil {
		c.log.Errorf("InventoryClient: Failed to marshal update stock data for ID %d: %v", productID, err)
		return fmt.Errorf("failed to prepare inventory update data: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.log.Errorf("InventoryClient: Failed to create UpdateStock request for ID %d: %v", productID, err)
		return fmt.Errorf("failed to create inventory update request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Errorf("InventoryClient: Failed to execute UpdateStock request for ID %d (new stock %d): %v", productID, newStock, err)
		return fmt.Errorf("failed to communicate with inventory service for stock update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.log.Warnf("InventoryClient: Product with ID %d not found for stock update (status %d)", productID, resp.StatusCode)
		return fmt.Errorf("product with ID %d not found in inventory for update", productID)
	}
	if resp.StatusCode == http.StatusBadRequest {

		bodyBytes, _ := io.ReadAll(resp.Body)
		c.log.Warnf("InventoryClient: Bad request updating stock for ID %d (status %d). Response body: %s", productID, resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("invalid stock update request for product %d (check inventory service logs)", productID)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.log.Errorf("InventoryClient: UpdateStock request for ID %d failed with status %d. Response body: %s", productID, resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("inventory service returned status %d for stock update on product %d", resp.StatusCode, productID)
	}

	c.log.Infof("InventoryClient: Successfully updated stock for product ID %d to %d", productID, newStock)
	return nil
}
