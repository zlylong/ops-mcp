import React from 'react';
import ReactDOM from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Avatar, ConfigProvider, Dropdown, Space, Typography } from 'antd';
import type { MenuProps } from 'antd';
import { ApiOutlined, UserOutlined } from '@ant-design/icons';
import zhCN from 'antd/locale/zh_CN';
import enUS from 'antd/locale/en_US';
import 'antd/dist/reset.css';
import './styles.css';

import { Dashboard } from './pages/Dashboard';

type Language = 'en' | 'zh';
let currentLanguage: Language = (localStorage.getItem('ops-mcp-language') as Language) || 'en';

const queryClient = new QueryClient({ defaultOptions: { queries: { refetchOnWindowFocus: false, retry: 1 } } });
const APP_VERSION = import.meta.env.VITE_APP_VERSION || '0.1.0';

function UserMenu() {
  const items: MenuProps['items'] = [
    {
      key: 'swagger',
      icon: <ApiOutlined />,
      label: 'Swagger API 文档',
      onClick: () => window.open('/swagger', '_blank', 'noopener,noreferrer'),
    },
  ];

  return (
    <div className="user-menu">
      <Dropdown menu={{ items }} placement="bottomRight" trigger={['click']}>
        <Space className="user-menu-trigger" size={8}>
          <Avatar size="small" icon={<UserOutlined />} />
          <Typography.Text strong>管理员</Typography.Text>
        </Space>
      </Dropdown>
    </div>
  );
}

function AppVersion() {
  return <div className="app-version">Ops MCP v{APP_VERSION}</div>;
}

function App() {
  const [language, setLanguage] = React.useState<Language>(currentLanguage);
  React.useEffect(() => { localStorage.setItem('ops-mcp-language', language); }, [language]);
  
  const locale = language === 'zh' ? zhCN : enUS;
  const theme = { token: { colorPrimary: '#1677ff', borderRadius: 10 } };

  return (
    <ConfigProvider locale={locale} theme={theme}>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <UserMenu />
          <AppVersion />
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="*" element={<div>Page not found</div>} />
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </ConfigProvider>
  );
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
