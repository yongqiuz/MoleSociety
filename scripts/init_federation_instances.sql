INSERT INTO federation_instances(name, focus, members, latency, status)
VALUES
  ('摩尔1号', '摩尔首服', '0 人在线', '未探测', '运行中'),
  ('摩尔2号', '摩尔的第二个联邦实例', '0 人在线', '未探测', '运行中'),
  ('摩尔3号', '摩尔的第三个联邦实例', '0 人在线', '未探测', '运行中')
ON CONFLICT (name) DO UPDATE
SET
  focus = EXCLUDED.focus,
  members = EXCLUDED.members,
  latency = EXCLUDED.latency,
  status = EXCLUDED.status;
