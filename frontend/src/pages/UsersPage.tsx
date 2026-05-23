import React, { useState } from 'react';
import {
  Table, Card, Button, Modal, Form, Input, Select, Tag, Space, Popconfirm,
  message, Typography, Tooltip,
} from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined, KeyOutlined } from '@ant-design/icons';
import { api } from '../services/api';
import type { User, UserCreateRequest, UserUpdateRequest } from '../types';

const { Text } = Typography;

export function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [pwdResetOpen, setPwdResetOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [messageApi, contextHolder] = message.useMessage();
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();
  const [pwdForm] = Form.useForm();

  const fetchUsers = async () => {
    setLoading(true);
    try {
      setUsers(await api.listUsers());
    } catch {
      messageApi.error('加载用户列表失败');
    } finally {
      setLoading(false);
    }
  };

  React.useEffect(() => {
    fetchUsers();
  }, []);

  const handleCreate = async (values: UserCreateRequest) => {
    setSubmitting(true);
    try {
      const newUser = await api.createUser(values);
      setUsers((prev) => [newUser, ...prev]);
      setCreateOpen(false);
      form.resetFields();
      messageApi.success('用户创建成功');
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      if (msg.includes('already taken')) {
        messageApi.error('用户名已被占用');
      } else {
        messageApi.error('创建失败');
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleEdit = async (values: UserUpdateRequest) => {
    if (!selectedUser) return;
    setSubmitting(true);
    try {
      const updated = await api.updateUser(selectedUser.id, values);
      setUsers((prev) => prev.map((u) => (u.id === selectedUser.id ? updated : u)));
      setEditOpen(false);
      messageApi.success('用户信息已更新');
    } catch {
      messageApi.error('更新失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await api.deleteUser(id);
      setUsers((prev) => prev.filter((u) => u.id !== id));
      messageApi.success('用户已删除');
    } catch {
      messageApi.error('删除失败');
    }
  };

  const handleResetPassword = async (values: { newPassword: string }) => {
    if (!selectedUser) return;
    setSubmitting(true);
    try {
      await api.resetUserPassword(selectedUser.id, { newPassword: values.newPassword });
      setPwdResetOpen(false);
      pwdForm.resetFields();
      messageApi.success('密码已重置');
      messageApi.info(`新密码：${values.newPassword}`);
    } catch {
      messageApi.error('密码重置失败');
    } finally {
      setSubmitting(false);
    }
  };

  const openEdit = (user: User) => {
    setSelectedUser(user);
    editForm.setFieldsValue({ nickname: user.nickname, email: user.email, status: user.status, role: user.role });
    setEditOpen(true);
  };

  const openResetPassword = (user: User) => {
    setSelectedUser(user);
    setPwdResetOpen(true);
  };

  const roleTag = (role: string) => {
    if (role === 'admin') return <Tag color="red">管理员</Tag>;
    if (role === 'operator') return <Tag color="orange">操作员</Tag>;
    return <Tag color="blue">查看者</Tag>;
  };

  const statusTag = (status: string) => (
    <Tag color={status === 'active' ? 'green' : 'default'}>{status === 'active' ? '正常' : '停用'}</Tag>
  );

  const columns = [
    { title: '用户名', dataIndex: 'username', key: 'username', width: 140 },
    { title: '昵称', dataIndex: 'nickname', key: 'nickname', width: 120, render: (v: string) => v || '-' },
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 180, render: (v: string) => v || '-' },
    { title: '角色', dataIndex: 'role', key: 'role', width: 100, render: roleTag },
    { title: '状态', dataIndex: 'status', key: 'status', width: 80, render: statusTag },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 170, render: (v: string) => new Date(v).toLocaleString('zh-CN') },
    {
      title: '操作',
      key: 'actions',
      width: 180,
      render: (_: unknown, record: User) => (
        <Space size="small">
          <Tooltip title="编辑用户"><Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} /></Tooltip>
          <Tooltip title="重置密码"><Button size="small" icon={<KeyOutlined />} onClick={() => openResetPassword(record)} /></Tooltip>
          <Popconfirm title="确定删除该用户？" onConfirm={() => handleDelete(record.id)} okText="删除" cancelText="取消">
            <Tooltip title="删除用户">
              <Button size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      {contextHolder}
      <Card
        title="用户管理"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setCreateOpen(true); }}>
            新建用户
          </Button>
        }
      >
        <Table
          dataSource={users}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 20, showSizeChanger: false }}
          size="small"
        />
      </Card>

      {/* Create user modal */}
      <Modal
        title="新建用户"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={null}
      >
        <Form form={form} layout="vertical" onFinish={handleCreate} style={{ marginTop: 16 }}>
          <Form.Item name="username" label="用户名" rules={[{ required: true, min: 3, max: 32, message: '3-32个字符' }]}>
            <Input placeholder="登录用户名" />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, min: 8, max: 128, message: '至少8个字符' }]}>
            <Input.Password placeholder="初始密码（至少8个字符）" />
          </Form.Item>
          <Form.Item name="nickname" label="昵称">
            <Input placeholder="显示名称（可选）" />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input placeholder="邮箱地址（可选）" />
          </Form.Item>
          <Form.Item name="role" label="角色" rules={[{ required: true, message: '请选择角色' }]} initialValue="viewer">
            <Select>
              <Select.Option value="viewer">查看者</Select.Option>
              <Select.Option value="operator">操作员</Select.Option>
              <Select.Option value="admin">管理员</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={submitting}>创建用户</Button>
              <Button onClick={() => setCreateOpen(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Edit user modal */}
      <Modal
        title={`编辑用户：${selectedUser?.username ?? ''}`}
        open={editOpen}
        onCancel={() => setEditOpen(false)}
        footer={null}
      >
        <Form form={editForm} layout="vertical" onFinish={handleEdit} style={{ marginTop: 16 }}>
          <Form.Item name="nickname" label="昵称">
            <Input placeholder="显示名称" />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input placeholder="邮箱地址" />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="active">
            <Select>
              <Select.Option value="active">正常</Select.Option>
              <Select.Option value="inactive">停用</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="role" label="角色">
            <Select>
              <Select.Option value="viewer">查看者</Select.Option>
              <Select.Option value="operator">操作员</Select.Option>
              <Select.Option value="admin">管理员</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={submitting}>保存</Button>
              <Button onClick={() => setEditOpen(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Reset password modal */}
      <Modal
        title={`重置密码：${selectedUser?.username ?? ''}`}
        open={pwdResetOpen}
        onCancel={() => { setPwdResetOpen(false); pwdForm.resetFields(); }}
        footer={null}
      >
        <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
          此操作将强制将用户密码改为新密码，请将新密码告知用户。
        </Text>
        <Form form={pwdForm} layout="vertical" onFinish={handleResetPassword}>
          <Form.Item
            name="newPassword"
            label="新密码"
            rules={[{ required: true, min: 8, message: '至少8个字符' }]}
          >
            <Input.Password placeholder="新密码（至少8个字符）" />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="确认密码"
            dependencies={['newPassword']}
            rules={[
              { required: true, message: '请再次输入' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) return Promise.resolve();
                  return Promise.reject(new Error('两次密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password placeholder="再次输入新密码" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={submitting}>重置密码</Button>
              <Button onClick={() => { setPwdResetOpen(false); pwdForm.resetFields(); }}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}