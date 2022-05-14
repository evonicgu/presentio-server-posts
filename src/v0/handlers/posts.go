package handlers

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"presentio-server-posts/src/v0/models"
	"presentio-server-posts/src/v0/repo"
	"presentio-server-posts/src/v0/util"
	"strconv"
	"time"
)

type PostsHandler struct {
	PostsRepo repo.PostsRepo
}

func CreatePostsHandler(group *gin.RouterGroup, postsRepo repo.PostsRepo) {
	handler := PostsHandler{
		PostsRepo: postsRepo,
	}

	group.GET("/:id", handler.getPost)
	group.POST("/", handler.createPost)
	//group.GET("/recommended")
	group.DELETE("/:id", handler.deletePost)
	group.GET("/user/:id/:page", handler.getUserPosts)
	group.GET("/user/self/:page", handler.getUserPostsSelf)
}

func (h *PostsHandler) getPost(c *gin.Context) {
	token, err := util.ValidateAccessTokenHeader(c.GetHeader("Authorization"))

	if err != nil {
		c.Status(util.HandleTokenError(err))

		return
	}

	claims, ok := token.Claims.(*util.UserClaims)

	if !ok {
		c.Status(403)
		return
	}

	postId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.Status(404)

		return
	}

	post, err := h.PostsRepo.FindById(postId)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(404)
		} else {
			c.Status(500)
		}

		return
	}

	c.Header("Cache-Control", "public, max-age=18000")
	c.Header("Pragma", "")
	c.Header("Expires", "")

	c.JSON(200, gin.H{
		"own":  post.UserID == claims.ID,
		"post": post,
	})
}

type PostParams struct {
	Text         string
	Tags         []string
	SourceID     *int64
	SourceUserId *int64
}

func (h *PostsHandler) createPost(c *gin.Context) {
	var params PostParams

	err := c.ShouldBindJSON(&params)

	if err != nil {
		c.Status(400)
		return
	}

	token, err := util.ValidateAccessTokenHeader(c.GetHeader("Authorization"))

	if err != nil {
		c.Status(util.HandleTokenError(err))
		return
	}

	claims, ok := token.Claims.(*util.UserClaims)

	if !ok {
		c.Status(403)
		return
	}

	post := models.Post{
		UserID:       claims.ID,
		Text:         params.Text,
		CreatedAt:    time.Now(),
		SourceID:     params.SourceID,
		SourceUserID: params.SourceUserId,
	}

	err = h.PostsRepo.Create(&post)

	if err != nil {
		c.Status(500)
		return
	}

	c.Status(201)
}

func (h *PostsHandler) deletePost(c *gin.Context) {
	token, err := util.ValidateAccessTokenHeader(c.GetHeader("Authorization"))

	if err != nil {
		c.Status(util.HandleTokenError(err))
		return
	}

	postId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.Status(404)

		return
	}

	claims, ok := token.Claims.(*util.UserClaims)

	if !ok {
		c.Status(403)
		return
	}

	err = h.PostsRepo.DeleteWithGuard(postId, claims.ID)

	if err != nil {
		fmt.Println(err.Error())
		c.Status(500)
		return
	}

	c.Status(204)
}

func (h *PostsHandler) getUserPosts(c *gin.Context) {
	userId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.Status(404)

		return
	}

	h.doGetUserPosts(userId, c)
}

func (h *PostsHandler) getUserPostsSelf(c *gin.Context) {
	h.doGetUserPosts(-1, c)
}

func (h *PostsHandler) doGetUserPosts(userId int64, c *gin.Context) {
	token, err := util.ValidateAccessTokenHeader(c.GetHeader("Authorization"))

	if err != nil {
		c.Status(util.HandleTokenError(err))
		return
	}

	page, err := strconv.Atoi(c.Param("page"))

	if err != nil {
		c.Status(404)

		return
	}

	claims, ok := token.Claims.(*util.UserClaims)

	if !ok {
		c.Status(403)
		return
	}

	if userId == -1 {
		userId = claims.ID
	}

	posts, err := h.PostsRepo.GetUserPosts(userId, page)

	if err != nil {
		c.Status(500)
		return
	}

	cache := "18000"

	if userId == claims.ID {
		cache = "300"
	}

	c.Header("Cache", "public,max-age="+cache)
	c.Header("Pragma", "")
	c.Header("Expires", "")

	c.JSON(200, gin.H{
		"own":   userId == claims.ID,
		"posts": posts,
	})
}
