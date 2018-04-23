DROP TABLE device_alerts;
DROP TABLE device_metrics;
DROP TABLE devices;
DROP TABLE users;

CREATE TABLE users
(
  id    SERIAL PRIMARY KEY,
  name  VARCHAR(255),
  email VARCHAR(255) NOT NULL
);

CREATE TABLE devices
(
  id      SERIAL PRIMARY KEY,
  name    VARCHAR(255) NOT NULL,
  user_id INT          NOT NULL,

  CONSTRAINT devices_user_id_fk FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE device_metrics
(
  id          SERIAL PRIMARY KEY,
  device_id   INT NOT NULL,
  metric_1    INT,
  metric_2    INT,
  metric_3    INT,
  metric_4    INT,
  metric_5    INT,
  local_time  TIMESTAMP, -- Время метрик на устройстве
  server_time TIMESTAMP DEFAULT NOW(), -- Серверноевремя сохранения метрик

  CONSTRAINT device_metrics_device_id_fk FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);

CREATE TABLE device_alerts
(
  id        SERIAL PRIMARY KEY,
  device_id INT,
  message   TEXT
);