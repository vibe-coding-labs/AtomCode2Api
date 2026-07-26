import React, { useEffect, useState } from 'react';
import { Suspense, lazy } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useSearchParams, useNavigate } from 'react-router-dom';
import { ConfigProvider, Spin, message } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import MainLayout from './layouts/MainLayout';
import Login from './pages/Login';
import Setup from './pages/Setup';
import ForgotPassword from './pages/ForgotPassword';
import OAuthError from './pages/OAuthError';
import { authApi, isAuthenticated, api } from './api';

const Dashboard = lazy(() => import('./pages/Dashboard'));
const Accounts = lazy(() => import('./pages/Accounts'));
const AccountDetail = lazy(() => import('./pages/AccountDetail'));
const Settings = lazy(() => import('./pages/Settings'));

const pageLoading = <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;

// OAuth callback handler — processes login_success / login_error from daemon OAuth redirect
const OAuthCallback: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  useEffect(() => {
    const loginSuccess = searchParams.get('login_success');
    const loginError = searchParams.get('login_error');

    if (loginSuccess) {
      message.success('OAuth 登录成功，正在导入账号...');
      api.autoLogin().then(() => {
        navigate('/accounts', { replace: true });
      }).catch(() => {
        navigate('/accounts', { replace: true });
      });
    } else if (loginError) {
      navigate('/oauth-error?error=' + encodeURIComponent(loginError), { replace: true });
    } else {
      navigate('/dashboard', { replace: true });
    }
  }, []);

  return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
};

const AuthGuard: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [checking, setChecking] = useState(true);
  const [initialized, setInitialized] = useState(true);
  const [authed, setAuthed] = useState(false);

  useEffect(() => {
    authApi.status().then((res) => {
      setInitialized(res.initialized);
      if (res.initialized) {
        setAuthed(isAuthenticated());
      }
      setChecking(false);
    }).catch(() => {
      setChecking(false);
    });
  }, []);

  if (checking) return pageLoading;

  if (!initialized) {
    return <Navigate to="/setup" replace />;
  }

  if (!authed) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

const App: React.FC = () => (
  <ConfigProvider locale={zhCN} theme={{ token: { colorPrimary: '#e8a050' } }}>
    <BrowserRouter>
      <Routes>
        <Route path="/setup" element={<Setup />} />
        <Route path="/login" element={<Login />} />
        <Route path="/forgot-password" element={<ForgotPassword />} />
        <Route path="/oauth-error" element={<OAuthError />} />
        <Route element={<AuthGuard><MainLayout /></AuthGuard>}>
          <Route path="/dashboard" element={<Suspense fallback={pageLoading}><Dashboard /></Suspense>} />
          <Route path="/accounts" element={<Suspense fallback={pageLoading}><Accounts /></Suspense>} />
          <Route path="/accounts/:userId" element={<Suspense fallback={pageLoading}><AccountDetail /></Suspense>} />
          <Route path="/settings" element={<Suspense fallback={pageLoading}><Settings /></Suspense>} />
        </Route>
        <Route path="/" element={<OAuthCallback />} />
      </Routes>
    </BrowserRouter>
  </ConfigProvider>
);

export default App;
