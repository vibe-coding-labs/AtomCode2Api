import React, { useEffect, useState } from 'react';
import { Layout, Menu, Typography, Tag, theme, Tooltip, Button, message } from 'antd';
import {
  DashboardOutlined,
  TeamOutlined,
  SettingOutlined,
  CheckCircleOutlined,
  LogoutOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import useDocumentTitle from '../hooks/useDocumentTitle';
import { api, clearToken } from '../api';

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

const menuItems = [
  { key: '/dashboard', icon: <DashboardOutlined />, label: '数据概览' },
  { key: '/accounts', icon: <TeamOutlined />, label: '账号管理' },
  { key: '/settings', icon: <SettingOutlined />, label: '系统设置' },
];

const COLLAPSED_KEY = 'atomcode_sider_collapsed';

const MainLayout: React.FC = () => {
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(COLLAPSED_KEY) === 'true');
  const navigate = useNavigate();
  const location = useLocation();
  const { token } = theme.useToken();
  useDocumentTitle();

  const [healthStatus, setHealthStatus] = useState<'ok' | 'error'>('ok');
  const [accountCount, setAccountCount] = useState(0);

  useEffect(() => {
    api.getHealth().then((h) => {
      setHealthStatus(h.status === 'ok' ? 'ok' : 'error');
      setAccountCount(h.accounts);
    }).catch(() => setHealthStatus('error'));
  }, []);

  const selectedKey = location.pathname.startsWith('/accounts') ? '/accounts'
    : location.pathname.startsWith('/settings') ? '/settings'
    : '/dashboard';

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={(val) => { setCollapsed(val); localStorage.setItem(COLLAPSED_KEY, String(val)); }}
        style={{ background: token.colorBgContainer }}
      >
        <div style={{
          height: 48,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
        }}>
          <img src="/favicon.ico" alt="AtomCode" style={{ width: 24, height: 24, marginRight: collapsed ? 0 : 8 }} />
          {!collapsed && <Text strong style={{ fontSize: 15 }}>AtomCode 2API</Text>}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{
          padding: '0 24px',
          background: token.colorBgContainer,
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Tag color={healthStatus === 'ok' ? 'success' : 'error'} icon={<CheckCircleOutlined />}>
              {healthStatus === 'ok' ? '服务正常' : '服务异常'}
            </Tag>
            <Text type="secondary">{accountCount} 个账号在线</Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Tooltip title="退出登录">
              <Button
                type="text"
                icon={<LogoutOutlined />}
                onClick={() => {
                  clearToken();
                  message.success('已退出登录');
                  window.location.href = '/login';
                }}
                style={{ color: token.colorTextSecondary }}
              />
            </Tooltip>
          </div>
        </Header>
        <Content style={{ margin: 24 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

export default MainLayout;