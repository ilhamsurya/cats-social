package handler

import (
	"fmt"
	"net/http"
	"projectsphere/cats-social/internal/cat/entity"
	"projectsphere/cats-social/internal/cat/service"
	"projectsphere/cats-social/pkg/middleware/auth"
	"projectsphere/cats-social/pkg/protocol/msg"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CatHandler struct {
	catSvc service.CatService
}

func NewCatHandler(catSvc service.CatService) CatHandler {
	return CatHandler{
		catSvc: catSvc,
	}
}
func (h CatHandler) Create(c *gin.Context) {

	if c.GetHeader("Authorization") == "" {
		c.JSON(http.StatusUnauthorized, msg.Unauthorization("No authorization header provided"))
		return
	}

	if c.Request.Body == nil {
		c.JSON(http.StatusBadRequest, msg.BadRequest("Request body is empty"))
		return
	}
	payload := new(entity.CatParam)
	err := c.ShouldBindJSON(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, msg.BadRequest(err.Error()))
		return
	}

	userID, err := auth.GetUserIdInsideCtx(c)
	if err != nil {
		fmt.Println(err)
	}

	if containsNull(payload) {
		c.JSON(http.StatusBadRequest, msg.BadRequest("JSON payload contains null values"))
		return
	}

	resp, err := h.catSvc.Create(c.Request.Context(), *payload, userID)
	if err != nil {
		respError := msg.UnwrapRespError(err)
		c.JSON(respError.Code, respError)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// containsNull checks if any field in the CatParam struct is null
func containsNull(param *entity.CatParam) bool {
	if param == nil {
		return false
	}

	// Check each field for null
	if param.Name == "" || param.Race == "" || param.AgeInMonth <= 0 {
		return true
	}
	return false
}

func (h CatHandler) Update(c *gin.Context) {
	payload := new(entity.CatParam)
	err := c.ShouldBindJSON(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, msg.BadRequest(err.Error()))
		return
	}
	catID := c.Param("id")

	if catID == "" {
		c.JSON(http.StatusBadRequest, msg.BadRequest("cat ID is required"))
		return
	}

	id, err := strconv.Atoi(catID)
	if err != nil {
		c.JSON(http.StatusNotFound, msg.BadRequest("id is not found"))
		return
	}

	resp, err := h.catSvc.Update(c.Request.Context(), id, *payload)
	fmt.Print(err)
	if err != nil {
		respError := msg.UnwrapRespError(err)
		c.JSON(respError.Code, respError)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h CatHandler) Get(c *gin.Context) {
	id := c.Query("id")
	limit := c.DefaultQuery("limit", "5")
	offset := c.DefaultQuery("offset", "0")
	race := c.Query("race")
	sex := c.Query("sex")
	hasMatched := c.Query("hasMatched")
	ageInMonth := c.Query("ageInMonth")
	owned := c.Query("owned")
	search := c.Query("search")

	idUser32, err := auth.GetUserIdInsideCtx(c)
	if err != nil {
		fmt.Println(err)
	}

	param := entity.GetCatParam{
		IdUser:     int(idUser32), //need get user id from jwt
		Race:       race,
		Sex:        sex,
		AgeInMonth: ageInMonth,
		Search:     search,
	}

	idParam, err := strconv.Atoi(id)
	if err == nil {
		param.IdCat = &idParam
	}

	limitParam, err := strconv.Atoi(limit)
	if err == nil {
		param.Limit = &limitParam
	}

	offsetParam, err := strconv.Atoi(offset)
	if err == nil {
		param.Offset = &offsetParam
	}

	hasMatchedParam, err := strconv.ParseBool(hasMatched)
	if err == nil {
		param.HasMatched = &hasMatchedParam
	}

	hasOwned, err := strconv.ParseBool(owned)
	if err == nil {
		param.Owned = &hasOwned
	}

	resp, err := h.catSvc.Get(c.Request.Context(), param)
	if err != nil {
		respError := msg.UnwrapRespError(err)
		c.JSON(respError.Code, respError)
		return
	}

	c.JSON(http.StatusOK, msg.ReturnResult("success", resp))
}

func (h CatHandler) Delete(c *gin.Context) {
	catID := c.Param("id")
	userID, err := auth.GetUserIdInsideCtx(c)
	if err != nil {
		fmt.Println(err)
	}

	if catID == "" {
		c.JSON(http.StatusBadRequest, msg.BadRequest("cat ID is required"))
		return
	}

	id, err := strconv.Atoi(catID)
	if err != nil {
		respError := msg.UnwrapRespError(msg.NotFound("id is not found"))
		c.JSON(respError.Code, respError)
		return
	}

	err = h.catSvc.Delete(c.Request.Context(), id, int(userID))
	if err != nil {
		respError := msg.UnwrapRespError(err)
		c.JSON(respError.Code, respError)
		return
	}

	c.JSON(http.StatusOK, msg.ReturnResult("successfully delete cat", nil))
}
