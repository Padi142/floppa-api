package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	pbToken     string
	pbTokenMu   sync.Mutex
	lastAuth    time.Time
	tokenExpiry = time.Hour
)

type pbAuthResponse struct {
	Token string `json:"token"`
}

func authenticatePB(pocketBaseURL, admin, password string) (string, error) {
	authURL := buildPBURL(pocketBaseURL, "/api/collections/_superusers/auth-with-password")

	body := map[string]string{
		"identity": admin,
		"password": password,
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", authURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyStr, _ := io.ReadAll(resp.Body)
		log.Printf("PocketBase auth failed: %s", string(bodyStr))
		return "", nil
	}

	var authResp pbAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}

	return authResp.Token, nil
}

func getPBToken(pocketBaseURL, admin, password string) string {
	pbTokenMu.Lock()
	defer pbTokenMu.Unlock()

	if pbToken != "" && time.Since(lastAuth) < tokenExpiry {
		return pbToken
	}

	token, err := authenticatePB(pocketBaseURL, admin, password)
	if err != nil || token == "" {
		log.Printf("Failed to authenticate with PocketBase: %v", err)
		return ""
	}

	pbToken = token
	lastAuth = time.Now()
	return pbToken
}

func buildPBURL(baseURL, path string) string {
	baseURL = strings.TrimSuffix(baseURL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return baseURL + path
}

func SetupPocketBaseProxy(r *gin.RouterGroup, pocketBaseURL, pocketBaseAdmin, pocketBasePassword string) {
	log.Printf("Setting up PocketBase proxy with URL: %s", pocketBaseURL)

	if pocketBaseAdmin != "" && pocketBasePassword != "" {
		log.Printf("Admin authentication configured for PocketBase proxy")
	}

	r.GET("/api/collections", func(c *gin.Context) {
		targetURL := buildPBURL(pocketBaseURL, "/api/collections")
		log.Printf("Proxying GET request to: %s", targetURL)

		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			log.Printf("Error creating request: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if pocketBaseAdmin != "" && pocketBasePassword != "" {
			token := getPBToken(pocketBaseURL, pocketBaseAdmin, pocketBasePassword)
			if token != "" {
				req.Header.Set("Authorization", token)
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("Error executing request to %s: %v", targetURL, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to connect to PocketBase: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		log.Printf("PocketBase response status: %d, content-type: %s, body length: %d", resp.StatusCode, resp.Header.Get("Content-Type"), len(body))
		if resp.StatusCode >= 400 {
			log.Printf("PocketBase error response body: %s", string(body))
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	})

	r.GET("/api/collections/:collection/records", func(c *gin.Context) {
		collection := c.Param("collection")
		sort := c.DefaultQuery("sort", "")
		page := c.DefaultQuery("page", "")
		perPage := c.DefaultQuery("perPage", "")

		reqURL := buildPBURL(pocketBaseURL, "/api/collections/"+collection+"/records")
		params := []string{}
		if sort != "" {
			params = append(params, "sort="+url.QueryEscape(sort))
		}
		if page != "" {
			params = append(params, "page="+url.QueryEscape(page))
		}
		if perPage != "" {
			params = append(params, "perPage="+url.QueryEscape(perPage))
		}
		if len(params) > 0 {
			reqURL += "?" + strings.Join(params, "&")
		}

		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if pocketBaseAdmin != "" && pocketBasePassword != "" {
			token := getPBToken(pocketBaseURL, pocketBaseAdmin, pocketBasePassword)
			if token != "" {
				req.Header.Set("Authorization", token)
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	})

	r.POST("/api/collections/:collection/records", func(c *gin.Context) {
		collection := c.Param("collection")

		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		for key, values := range form.Value {
			for _, value := range values {
				writer.WriteField(key, value)
			}
		}

		for key, files := range form.File {
			for _, file := range files {
				part, err := writer.CreateFormFile(key, file.Filename)
				if err != nil {
					writer.Close()
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				src, err := file.Open()
				if err != nil {
					writer.Close()
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				io.Copy(part, src)
				src.Close()
			}
		}

		writer.Close()

		req, err := http.NewRequest("POST", buildPBURL(pocketBaseURL, "/api/collections/"+collection+"/records"), body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())

		if pocketBaseAdmin != "" && pocketBasePassword != "" {
			token := getPBToken(pocketBaseURL, pocketBaseAdmin, pocketBasePassword)
			if token != "" {
				req.Header.Set("Authorization", token)
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
	})

	r.DELETE("/api/collections/:collection/records/:id", func(c *gin.Context) {
		collection := c.Param("collection")
		id := c.Param("id")

		req, err := http.NewRequest("DELETE", buildPBURL(pocketBaseURL, "/api/collections/"+collection+"/records/"+id), nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if pocketBaseAdmin != "" && pocketBasePassword != "" {
			token := getPBToken(pocketBaseURL, pocketBaseAdmin, pocketBasePassword)
			if token != "" {
				req.Header.Set("Authorization", token)
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	})

	r.GET("/api/files/:collection/:recordId/:filename", func(c *gin.Context) {
		collection := c.Param("collection")
		recordId := c.Param("recordId")
		filename := c.Param("filename")

		fileURL := buildPBURL(pocketBaseURL, "/api/files/"+collection+"/"+recordId+"/"+filename)

		req, err := http.NewRequest("GET", fileURL, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	})
}
