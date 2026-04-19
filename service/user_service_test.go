package service

import (
	"errors"
	"practice-8/repository"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetUserByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockUserRepository(ctrl)
	userService := NewUserService(mockRepo)

	user := &repository.User{ID: 1, Name: "Bakytzhan Agai"}
	mockRepo.EXPECT().GetUserByID(1).Return(user, nil)

	result, err := userService.GetUserByID(1)
	assert.NoError(t, err)
	assert.Equal(t, user, result)
}

func TestCreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockUserRepository(ctrl)
	userService := NewUserService(mockRepo)

	user := &repository.User{ID: 1, Name: "Bakytzhan Agai"}
	mockRepo.EXPECT().CreateUser(user).Return(nil)

	err := userService.CreateUser(user)
	assert.NoError(t, err)
}

func TestRegisterUser(t *testing.T) {
	t.Run("user already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockUserRepository(ctrl)
		userService := NewUserService(mockRepo)

		existingUser := &repository.User{ID: 1, Name: "Existing User"}
		user := &repository.User{ID: 2, Name: "New User"}

		mockRepo.EXPECT().GetByEmail("test@example.com").Return(existingUser, nil)

		err := userService.RegisterUser(user, "test@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user with this email already exists")
	})

	t.Run("new user success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockUserRepository(ctrl)
		userService := NewUserService(mockRepo)

		user := &repository.User{ID: 2, Name: "New User"}

		mockRepo.EXPECT().GetByEmail("newuser@example.com").Return(nil, nil)
		mockRepo.EXPECT().CreateUser(user).Return(nil)

		err := userService.RegisterUser(user, "newuser@example.com")
		assert.NoError(t, err)
	})

	t.Run("repository error on CreateUser", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockUserRepository(ctrl)
		userService := NewUserService(mockRepo)

		user := &repository.User{ID: 2, Name: "New User"}
		repoError := errors.New("database error")

		mockRepo.EXPECT().GetByEmail("newuser@example.com").Return(nil, nil)
		mockRepo.EXPECT().CreateUser(user).Return(repoError)

		err := userService.RegisterUser(user, "newuser@example.com")
		assert.Error(t, err)
		assert.Equal(t, repoError, err)
	})
}

func TestUpdateUserName(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockUserRepository(ctrl)
		userService := NewUserService(mockRepo)

		err := userService.UpdateUserName(1, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockUserRepository(ctrl)
		userService := NewUserService(mockRepo)

		repoError := errors.New("user not found")
		mockRepo.EXPECT().GetUserByID(999).Return(nil, repoError)

		err := userService.UpdateUserName(999, "New Name")
		assert.Error(t, err)
		assert.Equal(t, repoError, err)
	})

	t.Run("successful update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockUserRepository(ctrl)
		userService := NewUserService(mockRepo)

		user := &repository.User{ID: 1, Name: "Old Name"}
		mockRepo.EXPECT().GetUserByID(1).Return(user, nil)
		mockRepo.EXPECT().UpdateUser(user).Return(nil)

		err := userService.UpdateUserName(1, "New Name")
		assert.NoError(t, err)
		assert.Equal(t, "New Name", user.Name)
	})

	t.Run("UpdateUser fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockUserRepository(ctrl)
		userService := NewUserService(mockRepo)

		user := &repository.User{ID: 1, Name: "Old Name"}
		repoError := errors.New("update failed")
		mockRepo.EXPECT().GetUserByID(1).Return(user, nil)
		mockRepo.EXPECT().UpdateUser(user).Return(repoError)

		err := userService.UpdateUserName(1, "New Name")
		assert.Error(t, err)
		assert.Equal(t, repoError, err)
	})
}

func TestDeleteUser(t *testing.T) {
	t.Run("attempt to delete admin", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockUserRepository(ctrl)
		userService := NewUserService(mockRepo)

		err := userService.DeleteUser(1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "it is not allowed to delete admin user")
	})

	t.Run("successful delete", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockUserRepository(ctrl)
		userService := NewUserService(mockRepo)

		mockRepo.EXPECT().DeleteUser(2).Return(nil)

		err := userService.DeleteUser(2)
		assert.NoError(t, err)
	})

	t.Run("repository error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockUserRepository(ctrl)
		userService := NewUserService(mockRepo)

		repoError := errors.New("database error")
		mockRepo.EXPECT().DeleteUser(2).Return(repoError)

		err := userService.DeleteUser(2)
		assert.Error(t, err)
		assert.Equal(t, repoError, err)
	})
}
