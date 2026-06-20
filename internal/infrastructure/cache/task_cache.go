package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"taskmanager/internal/entity"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type TaskListCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewTaskListCache(client *redis.Client, ttl time.Duration) *TaskListCache {
	return &TaskListCache{client: client, ttl: ttl}
}

func (c *TaskListCache) Get(ctx context.Context, key string) ([]entity.Task, bool, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get tasks from cache")
	}

	var tasks []entity.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, false, errors.Wrap(err, "failed to unmarshal cached tasks")
	}

	return tasks, true, nil
}

func (c *TaskListCache) Set(ctx context.Context, key string, tasks []entity.Task) error {
	data, err := json.Marshal(tasks)
	if err != nil {
		return errors.Wrap(err, "failed to marshal tasks for cache")
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return errors.Wrap(err, "failed to set tasks to cache")
	}

	return nil
}

func (c *TaskListCache) InvalidateTeam(ctx context.Context, teamID int64) error {
	pattern := fmt.Sprintf("%s%d:*", teamCachePrefix, teamID)

	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return errors.Wrap(err, "failed to scan cache keys")
	}

	if len(keys) == 0 {
		return nil
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return errors.Wrap(err, "failed to delete cache keys")
	}

	return nil
}

const teamCachePrefix = "tasks:team:"

func TeamTasksKey(teamID int64, status string, assigneeID int64, limit, offset int) string {
	return fmt.Sprintf("%s%d:status=%s:assignee=%d:limit=%d:offset=%d",
		teamCachePrefix, teamID, status, assigneeID, limit, offset)
}
