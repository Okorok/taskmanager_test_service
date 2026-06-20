CREATE TABLE IF NOT EXISTS users (
    id            BIGINT       NOT NULL AUTO_INCREMENT,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(255) NOT NULL,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY users_email_uidx (email)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS teams (
    id         BIGINT       NOT NULL AUTO_INCREMENT,
    name       VARCHAR(255) NOT NULL,
    created_by BIGINT       NOT NULL,
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_teams_created_by FOREIGN KEY (created_by) REFERENCES users (id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS team_members (
    team_id   BIGINT      NOT NULL,
    user_id   BIGINT      NOT NULL,
    role      VARCHAR(32) NOT NULL,
    joined_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (team_id, user_id),
    CONSTRAINT fk_team_members_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE,
    CONSTRAINT fk_team_members_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    KEY team_members_user_idx (user_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS tasks (
    id          BIGINT       NOT NULL AUTO_INCREMENT,
    team_id     BIGINT       NOT NULL,
    title       VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL,
    status      VARCHAR(32)  NOT NULL DEFAULT 'todo',
    priority    VARCHAR(32)  NOT NULL DEFAULT 'medium',
    assignee_id BIGINT       NULL,
    created_by  BIGINT       NOT NULL,
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_tasks_team FOREIGN KEY (team_id) REFERENCES teams (id),
    CONSTRAINT fk_tasks_assignee FOREIGN KEY (assignee_id) REFERENCES users (id),
    CONSTRAINT fk_tasks_created_by FOREIGN KEY (created_by) REFERENCES users (id),
    KEY tasks_team_status_idx (team_id, status, id),
    KEY tasks_assignee_idx (assignee_id),
    KEY tasks_team_created_at_idx (team_id, created_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS task_history (
    id         BIGINT      NOT NULL AUTO_INCREMENT,
    task_id    BIGINT      NOT NULL,
    changed_by BIGINT      NOT NULL,
    field      VARCHAR(64) NOT NULL,
    old_value  TEXT        NULL,
    new_value  TEXT        NULL,
    changed_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_task_history_task FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_history_user FOREIGN KEY (changed_by) REFERENCES users (id),
    KEY task_history_task_idx (task_id, changed_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS task_comments (
    id         BIGINT   NOT NULL AUTO_INCREMENT,
    task_id    BIGINT   NOT NULL,
    user_id    BIGINT   NOT NULL,
    body       TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_task_comments_task FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_comments_user FOREIGN KEY (user_id) REFERENCES users (id),
    KEY task_comments_task_idx (task_id, created_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;
