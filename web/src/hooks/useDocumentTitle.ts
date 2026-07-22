import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

const TITLES: Record<string, string> = {
  '/': '数据概览 — AtomCode 2API',
  '/accounts': '账号管理 — AtomCode 2API',
  '/settings': '系统设置 — AtomCode 2API',
};

const DEFAULT_TITLE = 'AtomCode 2API';

const useDocumentTitle = () => {
  const location = useLocation();
  useEffect(() => {
    if (location.pathname.startsWith('/accounts/')) {
      const key = decodeURIComponent(location.pathname.replace('/accounts/', ''));
      document.title = `${key} — 账号详情 — AtomCode 2API`;
    } else {
      document.title = TITLES[location.pathname] || DEFAULT_TITLE;
    }
  }, [location.pathname]);
};

export default useDocumentTitle;
