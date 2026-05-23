import React, { useState } from 'react';
import { Card, Descriptions, Form, Input, Button, message, Divider, Tag, Space } from 'antd';
import { UserOutlined, LockOutlined, SaveOutlined } from '@ant-design/icons';
import { api } from '../services/api';
import type { User } from '../types';

export function UserProfilePage() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(false);
  const [pwdLoading, setPwdLoading] = useState(false);
  const [fetched, setFetched] = useState(false);
  const [messageApi, contextHolder] = message.useMessage();

  const fetchProfile = async () => {
    setLoading(true);
    try {
      const u = await api.getMe();
      setUser(u);
      setFetched(true);
    } catch {
      messageApi.error('获取个人信息失败');
    } finally {
      setLoading(false);
    }
  };

  React.useEffect(() => {
    if (!fetched) fetchProfile();
  }, []);

  const handleProfileUpdate = async (values: { nickname?: string; email?: string }) => {
    try {
      const updated = await api.updateMe(values);
      setUser(updated);
      messageApi.success('个人信息已更新');
    } catch {
      messageApi.error('更新失败');
    }
  };

  const handlePasswordChange = async (values: { oldPassword: string; newPassword: string; confirmPassword: string }) => {
    if (values.newPassword !== values.confirmPassword) {
      messageApi.error('两次输入的密码不一致');
      return;
    }
    setPwdLoading(true);
    try {
      await api.changeMyPassword({ oldPassword: values.oldPassword, newPassword: values.newPassword });
      messageApi.success('密码修改成功');
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      if (msg.includes('old password')) {
        messageApi.error('旧密码不正确');
      } else {
        messageApi.error('密码修改失败');
      }
    } finally {
      setPwdLoading(false);
    }
  };

  const roleTagColor = (role: string) => {
    if (role === 'admin') return 'red';
    if (role === 'operator') return 'orange';
    return 'blue';
  };

  return (
    <div style={{ padding: 24, maxWidth: 720 }}>
      {contextHolder}
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* Profile card */}
        <Card
          title={<><UserOutlined /> 个人信息</>}
          loading={loading}
          extra={user ? <Tag color={user.status === 'active' ? 'green' : 'default'}>{user.status === 'active' ? '正常' : '停用'}</Tag> : null}
        >
          {user ? (
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="用户名">{user.username}</Descriptions.Item>
              <Descriptions.Item label="角色">
                <Tag color={roleTagColor(user.role)}>{user.role === 'admin' ? '管理员' : user.role === 'operator' ? '操作员' : '查看者'}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="昵称">{user.nickname || '-'}</Descriptions.Item>
              <Descriptions.Item label="邮箱">{user.email || '-'}</Descriptions.Item>
              <Descriptions.Item label="创建时间" span={2}>{new Date(user.createdAt).toLocaleString('zh-CN')}</Descriptions.Item>
            </Descriptions>
          ) : null}
        </Card>

        {/* Edit profile form */}
        <Card title="修改个人信息" size="small">
          <Form
            layout="vertical"
            initialValues={{ nickname: user?.nickname ?? '', email: user?.email ?? '' }}
            onFinish={handleProfileUpdate}
          >
            <Form.Item label="昵称" name="nickname" rules={[{ min: 1, max: 64 }]}>
              <Input placeholder="输入昵称" />
            </Form.Item>
            <Form.Item label="邮箱" name="email" rules={[{ type: 'email', message: '请输入有效邮箱' }]}>
              <Input placeholder="输入邮箱" />
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" icon={<SaveOutlined />}>
                保存修改
              </Button>
            </Form.Item>
          </Form>
        </Card>

        {/* Change password form */}
        <Card title={<><LockOutlined /> 修改密码</>} size="small">
          <Form
            layout="vertical"
            onFinish={handlePasswordChange}
          >
            <Form.Item label="旧密码" name="oldPassword" rules={[{ required: true, message: '请输入旧密码' }]}>
              <Input.Password placeholder="输入旧密码" />
            </Form.Item>
            <Form.Item
              label="新密码"
              name="newPassword"
              rules={[
                { required: true, message: '请输入新密码' },
                { min: 8, message: '密码至少8个字符' },
              ]}
            >
              <Input.Password placeholder="输入新密码（至少8个字符）" />
            </Form.Item>
            <Form.Item
              label="确认新密码"
              name="confirmPassword"
              dependencies={['newPassword']}
              rules={[
                { required: true, message: '请再次输入新密码' },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue('newPassword') === value) return Promise.resolve();
                    return Promise.reject(new Error('两次输入的密码不一致'));
                  },
                }),
              ]}
            >
              <Input.Password placeholder="再次输入新密码" />
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" icon={<LockOutlined />} loading={pwdLoading}>
                修改密码
              </Button>
            </Form.Item>
          </Form>
        </Card>
      </Space>
    </div>
  );
}