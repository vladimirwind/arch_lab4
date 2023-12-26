package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

func SaveUserId(client redis.Client, usr *User) error {
	string_id := fmt.Sprintf("%d", usr.ID)
	var data User
	data.Name = usr.Name
	data.Surname = usr.Surname
	data.Login = usr.Login
	data.Password = usr.Password
	data.ID = usr.ID
	json_data, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error saving data to Redis: %w", err)
	}
	err = client.Set(string_id, json_data, 50*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("error saving data to Redis: %w", err)
	}

	return nil
}
func SaveUserLogin(client redis.Client, usr *User) error {
	json_data, err := json.Marshal(usr)
	if err != nil {
		return fmt.Errorf("error saving data to Redis: %w", err)
	}
	err = client.Set(usr.Login, json_data, 50*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("error saving data to Redis: %w", err)
	}
	return nil
}
func UserFromCacheId(client redis.Client, id string) string {
	data := client.Get(id)
	if len(data.Val()) < 1 {
		return ""
	}
	return data.Val()
}
func UserFromCacheLogin(client redis.Client, login string) string {
	data := client.Get(login)
	if len(data.Val()) < 1 {
		return ""
	}
	return data.Val()
}
