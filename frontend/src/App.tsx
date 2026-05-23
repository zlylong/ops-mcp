import React from 'react';
import ReactDOM from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom';
import { Avatar, ConfigProvider, Dropdown, Layout, Menu, Space, Typography } from 'antd';
import type { MenuProps } from 'antd';
import { ApiOutlined, AuditOutlined, CheckCircleOutlined, DashboardOutlined, FileSearchOutlined, KeyOutlined, ToolOutlined, UserOutlined } from '@ant-design/icons';
import zhCN from 'antd/locale/zh_CN';
import enUS from 'antd/locale/en_US';
import 'antd/dist/reset.css';
import './styles.css';

import { Dashboard } from './pages/Dashboard';
import { ToolsPage } from './pages/ToolsPage';
import { ExecutionsPage } from './pages/ExecutionsPage';
import { AuditPage } from './pages/AuditPage';
import { ApprovalsPage } from './pages/ApprovalsPage';
import { ToolApplicationsPage } from './pages/ToolApplicationsPage';
import { AgentAPIKeysPage } from './pages/AgentAPIKeysPage';
import { LoginPage } from './pages/LoginPage';
import { UserProfilePage } from './pages/UserProfilePage';
import { UsersPage } from './pages/UsersPage';

type Language = 'en' | 'zh';
let currentLanguage: Language = (localStorage.getItem('darwin-ops-mcp-language') as Language) || 'en';

const queryClient = new QueryClient({ defaultOptions: { queries: { refetchOnWindowFocus: false, retry: 1 } } });
const APP_VERSION = import.meta.env.VITE_APP_VERSION || '0.1.0';

function UserMenu() {
  const navigate = useNavigate();
  const items: MenuProps['items'] = [
    { key: 'profile', icon: <UserOutlined />, label: '个人中心', onClick: () => navigate('/profile') },
    { key: 'users', icon: <ApiOutlined />, label: '用户管理', onClick: () => navigate('/users') },
    { type: 'divider' as const },
    { key: 'swagger', icon: <ApiOutlined />, label: 'Swagger API 文档', onClick: () => window.open('/swagger', '_blank', 'noopener,noreferrer') },
  ];
  return (
    <Dropdown menu={{ items }} placement="bottomRight" trigger={['click']}>
      <Space className="user-menu-trigger" size={8}>
        <Avatar size="small" icon={<UserOutlined />} />
        <Typography.Text strong>管理员</Typography.Text>
      </Space>
    </Dropdown>
  );
}

function AppVersion() { return <div className="app-version">Darwin Ops MCP v{APP_VERSION}</div>; }

const menuItems = [
  { key: '/dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
  { key: '/tools', icon: <ToolOutlined />, label: '工具中心' },
  { key: '/executions', icon: <FileSearchOutlined />, label: '执行中心' },
  { key: '/audit', icon: <AuditOutlined />, label: '审计中心' },
  { key: '/approvals', icon: <CheckCircleOutlined />, label: '任务审批中心' },
  { key: '/tool-applications', icon: <ToolOutlined />, label: '工具审批中心' },
  { key: '/profile', icon: <UserOutlined />, label: '个人中心' },
  { key: '/users', icon: <KeyOutlined />, label: '用户管理' },
  { key: '/agent-keys', icon: <KeyOutlined />, label: 'Agent Key 管理' },
];

function AppShell() {
  const navigate = useNavigate();
  const location = useLocation();
  const selected = menuItems.find((item) => location.pathname.startsWith(item.key))?.key ?? '/dashboard';
  return (
    <Layout className="app-shell">
      <Layout.Sider className="sidebar" breakpoint="lg" collapsedWidth="0">
        <div className="brand"><ApiOutlined /><span style={{ cursor: "pointer" }} onClick={() => navigate("/")}>Darwin Ops MCP</span></div>
        <Menu theme="dark" mode="inline" selectedKeys={[selected]} items={menuItems} onClick={({ key }) => navigate(key)} />
      </Layout.Sider>
      <Layout>
        <Layout.Header className="topbar">
          <div>
            <Typography.Title level={4} style={{ margin: 0 }}>运维控制台</Typography.Title>
            <Typography.Text type="secondary">工具执行、审批与审计的一站式入口</Typography.Text>
          </div>
          <UserMenu />
        </Layout.Header>
        <Layout.Content className="main-content">
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/tools" element={<ToolsPage />} />
            <Route path="/executions" element={<ExecutionsPage />} />
            <Route path="/audit" element={<AuditPage />} />
            <Route path="/approvals" element={<ApprovalsPage />} />
            <Route path="/tool-applications" element={<ToolApplicationsPage />} />
            <Route path="/login" element={<LoginPage />} />
            <Route path="/profile" element={<UserProfilePage />} />
            <Route path="/users" element={<UsersPage />} />
            <Route path="/agent-keys" element={<AgentAPIKeysPage />} />
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </Layout.Content>
      </Layout>
      <AppVersion />
    </Layout>
  );
}

function App() {
  const [language, setLanguage] = React.useState<Language>(currentLanguage);
  React.useEffect(() => { localStorage.setItem('darwin-ops-mcp-language', language); }, [language]);
  const locale = language === 'zh' ? zhCN : enUS;
  const theme = { token: { colorPrimary: '#1677ff', borderRadius: 10 } };
  void setLanguage;

  return (
    <ConfigProvider locale={locale} theme={theme}>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter><AppShell /></BrowserRouter>
      </QueryClientProvider>
    </ConfigProvider>
  );
}

ReactDOM.createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>);
