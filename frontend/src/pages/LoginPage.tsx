import React, { useState } from 'react';
import { Card, Form, Input, Button, Typography, message, App } from 'antd';
import { useNavigate } from 'react-router-dom';
import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { api, API_TOKEN_STORAGE_KEY } from '../services/api';

const { Title, Text } = Typography;

export function LoginPage() {
  const [loading, setLoading] = useState(false);
  const [messageApi, contextHolder] = message.useMessage();
  const navigate = useNavigate();

  const handleLogin = async (values: { username: string; password: string }) => {
    setLoading(true);
    try {
      const res = await api.login(values);
      localStorage.setItem(API_TOKEN_STORAGE_KEY, res.token);
      localStorage.setItem('darwin-ops-mcp-current-user', JSON.stringify(res.user));
      messageApi.success('登录成功');
      navigate('/profile');
    } catch {
      messageApi.error('用户名或密码错误');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    }}>
      {contextHolder}
      <Card style={{ width: 400, boxShadow: '0 4px 24px rgba(0,0,0,0.15)', borderRadius: 12 }} styles={{ body: { padding: 32 } }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <Title level={3}>运维控制台登录</Title>
          <Text type="secondary">输入用户名和密码以访问系统</Text>
        </div>
        <Form layout="vertical" onFinish={handleLogin} size="large">
          <Form.Item
            name="username"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input prefix={<UserOutlined />} placeholder="用户名" autoComplete="username" />
          </Form.Item>
          <Form.Item
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="密码" autoComplete="current-password" />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" loading={loading} block>
              登录
            </Button>
          </Form.Item>
        </Form>
        <div style={{ marginTop: 16, textAlign: 'center' }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            默认账号：admin / admin1234
          </Text>
        </div>
      </Card>
    </div>
  );
}