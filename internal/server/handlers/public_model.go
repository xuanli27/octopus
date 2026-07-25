package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xuanli27/octopus/internal/model"
	"github.com/xuanli27/octopus/internal/op"
	"github.com/xuanli27/octopus/internal/server/middleware"
	"github.com/xuanli27/octopus/internal/server/resp"
	"github.com/xuanli27/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/api/v1/public-models").
		Use(middleware.Auth()).
		AddRoute(router.NewRoute("", http.MethodGet).Handle(listPublicModels)).
		AddRoute(router.NewRoute("", http.MethodPost).Handle(createPublicModel).Use(middleware.RequireJSON())).
		AddRoute(router.NewRoute("/resolve", http.MethodPost).Handle(resolvePublicModels).Use(middleware.RequireJSON())).
		AddRoute(router.NewRoute("/:id", http.MethodPut).Handle(updatePublicModel).Use(middleware.RequireJSON())).
		AddRoute(router.NewRoute("/:id", http.MethodDelete).Handle(deletePublicModel))
}

func listPublicModels(c *gin.Context) {
	rows, err := op.PublicModelList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, rows)
}

func createPublicModel(c *gin.Context) {
	var req model.PublicModelCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.InvalidJSON(c)
		return
	}
	row, err := op.PublicModelCreate(&req, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.Success(c, row)
}

func updatePublicModel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req model.PublicModelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.InvalidJSON(c)
		return
	}
	req.ID = id
	row, err := op.PublicModelUpdate(&req, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.Success(c, row)
}

func deletePublicModel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := op.PublicModelDelete(id, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.Success(c, nil)
}

type resolvePublicModelsRequest struct {
	Upstreams []string `json:"upstreams"`
}

func resolvePublicModels(c *gin.Context) {
	var req resolvePublicModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.InvalidJSON(c)
		return
	}
	rows, err := op.PublicModelResolveBatch(req.Upstreams, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, rows)
}
