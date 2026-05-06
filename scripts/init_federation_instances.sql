INSERT INTO federation_instances(name, focus, members, latency, status)
VALUES
  ('摩尔1号', '主实例：承载核心社交流与账号体系', '0 人在线', '未探测', '运行中'),
  ('摩尔2号', '阅读实例：面向内容订阅与社区讨论', '0 人在线', '未探测', '运行中'),
  ('摩尔3号', '联邦实例：负责跨实例消息转发与协同', '0 人在线', '未探测', '运行中')
ON CONFLICT (name) DO UPDATE
SET
  focus = EXCLUDED.focus,
  members = EXCLUDED.members,
  latency = EXCLUDED.latency,
  status = EXCLUDED.status;
