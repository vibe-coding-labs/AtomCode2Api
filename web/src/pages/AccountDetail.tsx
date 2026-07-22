import React, { useEffect, useState } from 'react';
import {
  Card, Row, Col, Statistic, Typography, Spin, Tag, Select, Button,
  message, Space, Table, Badge, Segmented, Popconfirm, Tooltip, Progress,
} from 'antd';
import {
  ArrowLeftOutlined, ApiOutlined, ThunderboltOutlined,
  CheckCircleOutlined, ReloadOutlined,
  ClockCircleOutlined, GlobalOutlined, FireOutlined,
  DeleteOutlined, QuestionCircleOutlined,
  CloseCircleOutlined, SwapOutlined, EyeOutlined, 
  CopyOutlined, CodeOutlined, RobotOutlined, InfoCircleOutlined,
} from '@ant-design/icons';
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip as RTooltip,
  ResponsiveContainer, PieChart, Pie, Cell, AreaChart, Area,
} from 'recharts';
import { useParams, useNavigate } from 'react-router-dom';
import { api, accountDisplayName } from '../api';
import type { Account, AccountStats, ModelCatalogItem, RequestLog } from '../api';

const C = { accent: '#e8a050', success: '#69d28a', error: '#ff6e6e', warn: '#faad14' };
const PIE_COLORS = ['#e8a050', '#7aa6ff', '#69d28a', '#95de64', '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16'];

const latColor = (ms: number) => ms < 500 ? C.success : ms < 1500 ? C.warn : C.error;
const fmt = (n: number) => n >= 1_000_000 ? (n / 1_000_000).toFixed(2) + 'M' : n >= 1_000 ? (n / 1_000).toFixed(1) + 'K' : n.toLocaleString();
const stTag = (c: number) => <Tag color={c >= 200 && c < 300 ? 'success' : c >= 400 && c < 500 ? 'warning' : 'error'}>{c}</Tag>;
const ft = (t: string) => {
  if (!t) return '-';
  const d = new Date(t + (t.includes('Z') || t.includes('+') ? '' : 'Z'));
  if (isNaN(d.getTime())) return t;
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
};
const flat = (ms: number) => ms < 1000 ? `${ms}ms` : ms < 60000 ? `${Math.floor(ms / 1000)}s${ms % 1000}ms` : `${Math.floor(ms / 60000)}m${Math.floor((ms % 60000) / 1000)}s`;

const copy = async (text: string, msg: string) => {
  try { if (navigator.clipboard?.writeText) { await navigator.clipboard.writeText(text); message.success(msg); return; } } catch {}
  const ta = document.createElement('textarea');
  ta.value = text; ta.style.position = 'fixed'; ta.style.left = '-9999px'; ta.style.top = '-9999px';
  document.body.appendChild(ta); ta.select();
  try { document.execCommand('copy'); message.success(msg); } catch { message.error('复制失败'); }
  document.body.removeChild(ta);
};

const CodeBlock: React.FC<{ code: string; onCopy: () => void }> = ({ code, onCopy }) => (
  <div style={{ position: 'relative' }}>
    <pre style={{ background: '#1e1e1e', color: '#d4d4d4', padding: '8px 10px', margin: 0, fontSize: 11, lineHeight: 1.5, overflowX: 'auto', whiteSpace: 'pre', fontFamily: "'JetBrains Mono','Fira Code','Consolas',monospace" }}>
      <code>{code}</code>
    </pre>
    <Button type="text" icon={<CopyOutlined />} style={{ position: 'absolute', top: 1, right: 1, color: '#aaa', fontSize: 11 }} onClick={(e) => { e.stopPropagation(); onCopy(); }} />
  </div>
);

const Hint: React.FC<{ title: string }> = ({ title }) => (
  <Tooltip title={title}><InfoCircleOutlined style={{ color: '#bbb', cursor: 'help', fontSize: 10, marginLeft: 3 }} /></Tooltip>
);

const SECTION = { marginBottom: 8, borderRadius: 3 };


const AccountDetail: React.FC = () => {
  const { userId } = useParams<{ userId: string }>();
  const navigate = useNavigate();
  const [account, setAccount] = useState<Account | null>(null);
  const [stats, setStats] = useState<AccountStats | null>(null);
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [catalog, setCatalog] = useState<ModelCatalogItem[]>([]);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [savingModel, setSavingModel] = useState(false);
  const [logFilter, setLogFilter] = useState<string>('all');
  const [activeSessions, setActiveSessions] = useState(0);
  const [cpStatus, setCpStatus] = useState<any>(null);
  const key = userId ? decodeURIComponent(userId) : '';

  // Derive last active time from latest log
  const lastActive = logs.length > 0 ? logs[0].created_at : null;
  const isActiveRecently = lastActive ? (Date.now() - new Date(lastActive + (lastActive.includes('Z') || lastActive.includes('+') ? '' : 'Z')).getTime()) < 24 * 3600 * 1000 : false;

  const fetchData = async () => {
    setLoading(true);
    try {
      const [accs, ss, ls] = await Promise.all([api.listAccounts(), api.getAccountStats(key), api.getAccountLogs(key, 500)]);
      setAccount(accs.find(a => a.user_id === key) || null);
      setStats(ss); setLogs(ls.logs || []);
    } catch (e: unknown) { message.error(e instanceof Error ? e.message : '加载失败'); } finally { setLoading(false); }
  };
  const fetchCatalog = async () => { setCatalogLoading(true); try { setCatalog(await api.listModelsCatalog()); } catch {} finally { setCatalogLoading(false); } };
  const fetchCp = async () => { try { setCpStatus(await api.getCodingPlanStatus()); } catch {} };

  useEffect(() => { fetchData(); fetchCatalog(); fetchCp(); }, [key]);

  useEffect(() => {
    const poll = async () => { try { const acc = (await api.listAccounts()).find(a => a.user_id === key); if (acc) setActiveSessions(acc.active_sessions); } catch {} };
    poll(); const id = setInterval(poll, 5000); return () => clearInterval(id);
  }, [key]);

  const handleModelChange = async (m: string) => {
    setSavingModel(true);
    try { await api.updateAccountModel(key, m); message.success(`默认模型已更新`); fetchData(); } catch (e: unknown) { message.error(e instanceof Error ? e.message : '更新失败'); } finally { setSavingModel(false); }
  };

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  if (!account) return <div style={{ textAlign: 'center', padding: 100 }}>账号不存在</div>;

  const modelOpts = catalog.filter(m => m.id !== 'test').map(m => ({ label: `${m.name} · ${m.free ? '免费' : m.input_price}`, value: m.id }));
  const pric = (r: ModelCatalogItem) => r.free
    ? <Tag color="success" style={{ fontSize: 10, lineHeight: '16px' }}>免费</Tag>
    : <div><Tag color="warning" style={{ fontSize: 10, lineHeight: '16px', marginBottom: 1 }}>付费</Tag><div style={{ fontSize: 10, color: '#666' }}>入 {r.input_price}<br />出 {r.output_price}</div></div>;

  const fLogs = logFilter === 'all' ? logs : logFilter === 'errors' ? logs.filter(l => l.status_code >= 400) : logs.filter(l => l.stream);
  const epData = stats?.by_endpoint.map(e => ({ name: e.endpoint.replace('/v1/', ''), value: e.count })) || [];

  const logCols = [
    { title: '时间', dataIndex: 'created_at', key: 't', width: 140, render: (t: string) => <Typography.Text style={{ fontSize: 10, fontFamily: 'monospace' }}>{ft(t)}</Typography.Text> },
    { title: '端点', dataIndex: 'endpoint', key: 'e', width: 160, render: (ep: string) => <Typography.Text code style={{ fontSize: 10 }}>{ep}</Typography.Text> },
    { title: '模型', dataIndex: 'model', key: 'm', width: 120, ellipsis: true, render: (m: string) => m || '-' },
    { title: '流', dataIndex: 'stream', key: 's', width: 30, render: (s: boolean) => <Badge status={s ? 'processing' : 'default'} /> },
    { title: '状态', dataIndex: 'status_code', key: 'st', width: 50, render: (c: number) => stTag(c) },
    { title: '入', dataIndex: 'input_tokens', key: 'i', width: 55, sorter: (a: RequestLog, b: RequestLog) => a.input_tokens - b.input_tokens, render: (n: number) => <Typography.Text style={{ fontSize: 10, fontFamily: 'monospace' }}>{n > 0 ? fmt(n) : '-'}</Typography.Text> },
    { title: '出', dataIndex: 'output_tokens', key: 'o', width: 55, sorter: (a: RequestLog, b: RequestLog) => a.output_tokens - b.output_tokens, render: (n: number) => <Typography.Text style={{ fontSize: 10, fontFamily: 'monospace' }}>{n > 0 ? fmt(n) : '-'}</Typography.Text> },
    { title: '延迟', dataIndex: 'latency_ms', key: 'l', width: 70, sorter: (a: RequestLog, b: RequestLog) => a.latency_ms - b.latency_ms, render: (ms: number) => <Typography.Text style={{ color: latColor(ms), fontFamily: 'monospace', fontWeight: 500, fontSize: 10 }}>{flat(ms)}</Typography.Text> },
  ];

  return (
    <div style={{ padding: '0 4px' }}>
      {/* ═══ Row 1: Account info (compact, no CodingPlan) ═══ */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 8, flexWrap: 'wrap', alignItems: 'flex-start' }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/accounts')} type="text" style={{ flexShrink: 0, marginTop: 2, fontSize: 13, width: 24, height: 24 }} />
        <div style={{ minWidth: 0, flex: 1 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
            <Typography.Title level={5} style={{ margin: 0, fontSize: 15, lineHeight: '22px' }}>{accountDisplayName(account)}</Typography.Title>
            {account.is_default && <Tag color="blue" style={{ fontSize: 10, lineHeight: '16px' }}>默认</Tag>}
            {activeSessions > 0 && <Tag color="processing" style={{ fontSize: 10, lineHeight: '16px' }}>{activeSessions} 活跃</Tag>}
            {logs.length > 0 && (isActiveRecently
              ? <Tag color="success" style={{ fontSize: 10, lineHeight: '16px' }}>近 24h 活跃</Tag>
              : <Tag color="warning" style={{ fontSize: 10, lineHeight: '16px' }}>超过 24h 未使用</Tag>
            )}
          </div>
          <div style={{ fontSize: 10, color: '#999', marginTop: 1, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <span>ID: {key}<Hint title="AtomCode 用户唯一标识" /></span>
            <span>创建: {account.created_at?.slice(0, 10) || '-'}</span>
            <span>
              上次活跃:
              {logs.length > 0
                ? <span style={{ color: isActiveRecently ? C.success : '#999' }}> {ft(logs[0].created_at)}</span>
                : <span style={{ color: '#999' }}> 暂无记录</span>
              }
              <Hint title="最近一次 API 请求的时间。如果超过 24 小时无请求，可能表示账号处于闲置状态" />
            </span>
          </div>
          <div style={{ marginTop: 2, display: 'flex', alignItems: 'center', gap: 4 }}>
            <Typography.Text style={{ fontSize: 10, color: '#666', whiteSpace: 'nowrap' }}>Token<Hint title="客户端连接密钥，设置 ANTHROPIC_API_KEY 或 OPENAI_API_KEY" />:</Typography.Text>
            <div style={{ maxWidth: 360, overflowX: 'auto' }}>
              <Typography.Text code copyable style={{ fontSize: 9, whiteSpace: 'nowrap', display: 'inline-block' }}>{account.api_token}</Typography.Text>
            </div>
          </div>
        </div>
      </div>

      {/* ═══ Row 2: CodingPlan full-width card — compact vertical ═══ */}
      {cpStatus && (
        <Card size="small" style={{ marginBottom: 8, borderRadius: 2, border: '1px solid #e8e8e8', background: 'linear-gradient(135deg, #fafafa 0%, #fff 100%)' }} styles={{ body: { padding: '10px 14px' } }}>
          <div style={{ display: 'flex', gap: 14, alignItems: 'center', flexWrap: 'wrap' }}>
            <Progress type="circle" size={48} percent={Math.min(cpStatus.usage.current_window_percent, 100)} strokeColor={cpStatus.usage.current_window_percent > 80 ? C.error : C.accent} format={() => `${cpStatus.usage.current_window_percent}%`} strokeWidth={7} />
            <div style={{ flex: 1, minWidth: 180 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap', marginBottom: 2 }}>
                <Typography.Text strong style={{ fontSize: 14 }}>{cpStatus.plan.name}</Typography.Text>
                <Tag color="success" style={{ fontSize: 9, lineHeight: '14px' }}>使用中</Tag>
                <Hint title="AtomCode 套餐。Lite 免费，Pro 付费" />
              </div>
              <div style={{ fontSize: 12, color: '#666', lineHeight: 1.6 }}>
                到期 {cpStatus.plan.expires_at} · 剩 {cpStatus.plan.remaining_days}d / 共 {cpStatus.plan.total_days}d · 重置 {cpStatus.usage.resets_at}
              </div>
              <div style={{ marginTop: 3, display: 'flex', gap: 10, flexWrap: 'wrap' }}>
                <div>
                  <span style={{ fontSize: 10, color: '#666' }}>免费 <Hint title="CodingPlan 免费覆盖" /></span>
                  {cpStatus.free_models.map((m: string) => <Tag key={m} color="success" style={{ fontSize: 9, lineHeight: '14px', marginLeft: 3 }}>{m.split('/').pop()}</Tag>)}
                </div>
                {cpStatus.paid_models?.length > 0 && (
                  <div>
                    <span style={{ fontSize: 10, color: '#666' }}>付费 <Hint title="需升级 Pro 或自行购买" /></span>
                    {cpStatus.paid_models.map((m: string) => <Tag key={m} color="warning" style={{ fontSize: 9, lineHeight: '14px', marginLeft: 3 }}>{m.split('/').pop()}</Tag>)}
                  </div>
                )}
              </div>
            </div>
          </div>
          {cpStatus.note && (
            <div style={{ marginTop: 6, padding: '5px 8px', background: '#f6f8fa', borderRadius: 2, fontSize: 11, color: '#666', lineHeight: 1.5 }}>
              <InfoCircleOutlined style={{ marginRight: 4, color: C.accent }} />
              {cpStatus.note}
            </div>
          )}
        </Card>
      )}

      {/* ═══ Quick action bar ═══ */}
      <div style={{ marginBottom: 8, display: 'flex', gap: 6, flexWrap: 'wrap', alignItems: 'center' }}>
        <Select style={{ width: 200 }} value={account.default_model || undefined} placeholder="默认模型" options={modelOpts} allowClear loading={catalogLoading} onChange={handleModelChange} disabled={savingModel} size="small" />
        <Popconfirm title="重置 API Token？" description="重置后旧 Token 将立即失效，已配置的客户端需要更新 Token。" onConfirm={async () => { try { await api.renewToken(key); message.success('Token 已更新'); fetchData(); } catch (e: unknown) { message.error(e instanceof Error ? e.message : '更新失败'); } }} okText="确认重置" cancelText="取消">
          <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }}>重置 Token</Button>
        </Popconfirm>
        <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} icon={<ReloadOutlined />} onClick={() => { fetchData(); fetchCatalog(); fetchCp(); }}>刷新</Button>
        <Popconfirm title={`删除「${accountDisplayName(account)}」？`} description="关联客户端将无法访问" onConfirm={async () => { try { await api.removeAccount(key); message.success('已删除'); navigate('/accounts'); } catch (e: unknown) { message.error(e instanceof Error ? e.message : '删除失败'); } }}>
          <Button size="small" danger style={{ fontSize: 11, lineHeight: '18px' }} icon={<DeleteOutlined />}>删除</Button>
        </Popconfirm>
        <div style={{ flex: 1 }} />
        <Tooltip title="客户端指定模型优先，此处默认模型仅用于客户端未指定时"><QuestionCircleOutlined style={{ color: '#999', cursor: 'help', fontSize: 12 }} /></Tooltip>
      </div>

      {/* ═══ Quick Start — compact side by side ═══ */}
      <Card style={{ ...SECTION, overflow: 'hidden', border: '1px solid #e8e8e8' }} styles={{ body: { padding: 0 } }}>
        <div style={{ background: `linear-gradient(135deg, ${C.accent} 0%, #c87618 100%)`, padding: '6px 10px' }}>
          <Typography.Text strong style={{ color: '#fff', fontSize: 12 }}><CodeOutlined style={{ marginRight: 4 }} />快速接入</Typography.Text>
        </div>
        <div style={{ padding: '6px 8px' }}>
          <Row gutter={[6, 6]}>
            <Col xs={24} lg={12}>
              <Card size="small" style={{ borderRadius: 2 }} styles={{ body: { padding: '6px 8px' } }}
                title={<Space size={4}><RobotOutlined style={{ color: C.accent, fontSize: 12 }} /><span style={{ fontSize: 11 }}>Claude Code</span><Tag color="blue" style={{ fontSize: 9, lineHeight: '14px' }}>Anthropic</Tag></Space>}>
                <CodeBlock code={[
                  `ANTHROPIC_BASE_URL=http://192.168.1.90:13457 \\`,
                  `ANTHROPIC_API_KEY=${account.api_token} \\`,
                  `ANTHROPIC_MODEL=${account.default_model || 'deepseek-v4-flash'} \\`,
                  `CLAUDE_CODE_MAX_OUTPUT_TOKENS=65536 \\`,
                  `API_TIMEOUT_MS=6000000 \\`,
                  `CLAUDE_CODE_MAX_RETRIES=1000000 \\`,
                  `claude --dangerously-skip-permissions`,
                ].join('\n')} onCopy={() => copy(`ANTHROPIC_BASE_URL=http://192.168.1.90:13457\nANTHROPIC_API_KEY=${account.api_token}\nANTHROPIC_MODEL=${account.default_model || 'deepseek-v4-flash'}\nCLAUDE_CODE_MAX_OUTPUT_TOKENS=65536\nAPI_TIMEOUT_MS=6000000\nCLAUDE_CODE_MAX_RETRIES=1000000\nclaude --dangerously-skip-permissions`, 'Claude Code 命令已复制')} />
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card size="small" style={{ borderRadius: 2 }} styles={{ body: { padding: '6px 8px' } }}
                title={<Space size={4}><CodeOutlined style={{ color: '#722ed1', fontSize: 12 }} /><span style={{ fontSize: 11 }}>Codex / OpenAI</span><Tag style={{ fontSize: 9, lineHeight: '14px' }}>OpenAI</Tag></Space>}>
                <CodeBlock code={[
                  `OPENAI_BASE_URL=http://192.168.1.90:13457/v1 \\`,
                  `OPENAI_API_KEY=${account.api_token} \\`,
                  `OPENAI_MODEL=${account.default_model || 'deepseek-v4-flash'} \\`,
                  `NODE_TLS_REJECT_UNAUTHORIZED=0 \\`,
                  `API_TIMEOUT_MS=6000000 \\`,
                  `codex exec "你的问题"`,
                ].join('\n')} onCopy={() => copy(`OPENAI_BASE_URL=http://192.168.1.90:13457/v1\nOPENAI_API_KEY=${account.api_token}\nOPENAI_MODEL=${account.default_model || 'deepseek-v4-flash'}\nNODE_TLS_REJECT_UNAUTHORIZED=0\nAPI_TIMEOUT_MS=6000000\ncodex exec "你的问题"`, 'Codex 命令已复制')} />
              </Card>
            </Col>
          </Row>
        </div>
      </Card>

      {/* ═══ Model Catalog ═══ */}
      <Card title={<span style={{ fontSize: 12 }}><EyeOutlined style={{ marginRight: 4 }} />可用模型</span>} size="small" style={SECTION}
        extra={<Space size={4}><Tag color="success" style={{ fontSize: 9, lineHeight: '14px' }}>免费{catalog.filter(m => m.free && m.id !== 'test').length}</Tag><Tag color="warning" style={{ fontSize: 9, lineHeight: '14px' }}>付费{catalog.filter(m => !m.free && m.id !== 'test').length}</Tag></Space>}
        loading={catalogLoading}>
        <Table dataSource={catalog.filter(m => m.id !== 'test')} rowKey="id" size="small" pagination={false}
          columns={[
            { title: '模型', dataIndex: 'name', key: 'n', width: 170, render: (n: string, r: ModelCatalogItem) => <Space size={4}><Typography.Text strong style={{ fontSize: 11 }}>{n}</Typography.Text><Typography.Text code style={{ fontSize: 9 }}>{r.id}</Typography.Text></Space> },
            { title: '提供商', dataIndex: 'provider', key: 'p', width: 70, render: (v: string) => <Typography.Text style={{ fontSize: 11 }}>{v}</Typography.Text> },
            { title: '类型', dataIndex: 'type', key: 't', width: 55, render: (t: string) => <Tag color={{ chat: 'blue', reasoning: 'purple', vision: 'cyan' }[t] || 'default'} style={{ fontSize: 9, lineHeight: '14px' }}>{t}</Tag> },
            { title: '上下文', dataIndex: 'context_window', key: 'c', width: 90, render: (n: number) => <Typography.Text style={{ fontSize: 11 }}>{n >= 1000000 ? `${(n / 10000).toFixed(0)}万` : `${(n / 1000).toFixed(0)}K`}</Typography.Text> },
            { title: '定价', key: 'pr', width: 90, render: (_: unknown, r: ModelCatalogItem) => pric(r) },
            { title: '', key: 'def', width: 40, render: (_: unknown, r: ModelCatalogItem) => r.default ? <Tag color="blue" style={{ fontSize: 9, lineHeight: '14px' }}>默认</Tag> : null },
          ]} />
      </Card>

      {/* ═══ Stats + Charts ═══ */}
      {stats && (
        <>
          <Row gutter={[6, 6]} style={{ marginBottom: 8 }}>
            <Col xs={24} md={12}>
              <Card size="small" style={{ borderRadius: 3 }} styles={{ body: { padding: '6px 8px' } }}>
                <Row gutter={[4, 4]}>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>请求<Hint title="今日 API 请求总数" /></span>} value={stats.total_requests} valueStyle={{ fontSize: 16, color: C.accent }} prefix={<ApiOutlined style={{ fontSize: 12 }} />} /></Col>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>成功</span>} value={stats.success_count} valueStyle={{ fontSize: 16, color: C.success }} prefix={<CheckCircleOutlined style={{ fontSize: 12 }} />} /></Col>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>失败</span>} value={stats.error_count} valueStyle={{ fontSize: 16, color: stats.error_count > 0 ? C.error : C.success }} prefix={<CloseCircleOutlined style={{ fontSize: 12 }} />} /></Col>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>流式<Hint title="流式 SSE 请求数" /></span>} value={stats.stream_count} valueStyle={{ fontSize: 12 }} prefix={<SwapOutlined style={{ fontSize: 10 }} />} /></Col>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>延迟<Hint title="平均响应时间" /></span>} value={Math.round(stats.avg_latency_ms)} suffix="ms" valueStyle={{ fontSize: 12, color: stats.avg_latency_ms < 500 ? C.success : stats.avg_latency_ms < 1500 ? C.warn : C.error }} prefix={<ThunderboltOutlined style={{ fontSize: 10 }} />} /></Col>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>累计</span>} value={stats.all_time?.total_requests ?? 0} valueStyle={{ fontSize: 12 }} /></Col>
                </Row>
              </Card>
            </Col>
            <Col xs={24} md={12}>
              <Card size="small" style={{ borderRadius: 3 }} styles={{ body: { padding: '6px 8px' } }}>
                <Row gutter={[4, 4]}>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>Token<Hint title="今日输入+输出总量" /></span>} value={fmt(stats.total_input_tokens + stats.total_output_tokens)} valueStyle={{ fontSize: 16, color: C.accent }} prefix={<FireOutlined style={{ fontSize: 12 }} />} /></Col>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>输入</span>} value={fmt(stats.total_input_tokens)} valueStyle={{ fontSize: 12 }} /></Col>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>输出</span>} value={fmt(stats.total_output_tokens)} valueStyle={{ fontSize: 12 }} /></Col>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>累计 T</span>} value={fmt((stats.all_time?.total_input_tokens ?? 0) + (stats.all_time?.total_output_tokens ?? 0))} valueStyle={{ fontSize: 12 }} /></Col>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>均/请求<Hint title="每次请求平均 Token" /></span>} value={stats.total_requests > 0 ? fmt(Math.round((stats.total_input_tokens + stats.total_output_tokens) / stats.total_requests)) : '-'} valueStyle={{ fontSize: 12 }} /></Col>
                  <Col span={8}><Statistic title={<span style={{ fontSize: 10 }}>入/出比<Hint title="输入/输出 Token 比例" /></span>} value={stats.total_output_tokens > 0 ? (stats.total_input_tokens / stats.total_output_tokens).toFixed(1) : '-'} suffix={stats.total_output_tokens > 0 ? ':1' : ''} valueStyle={{ fontSize: 12 }} /></Col>
                </Row>
              </Card>
            </Col>
          </Row>

          {stats.hourly?.length > 0 && (() => {
            const hm = new Map(stats.hourly.map(h => [h.hour, h]));
            const now = new Date();
            const d = Array.from({ length: 24 }, (_, i) => {
              const dt = new Date(now.getTime() - i * 3600000);
              const k = `${String(dt.getMonth() + 1).padStart(2, '0')}-${String(dt.getDate()).padStart(2, '0')} ${String(dt.getHours()).padStart(2, '0')}`;
              const e = hm.get(k);
              return { label: `${String(dt.getHours()).padStart(2, '0')}:00`, count: e?.count ?? 0, input_tokens: e?.input_tokens ?? 0, output_tokens: e?.output_tokens ?? 0, errors: e?.errors ?? 0 };
            }).reverse();
            return (
              <Row gutter={[6, 6]} style={{ marginBottom: 8 }}>
                <Col xs={24} lg={12}><Card title={<span style={{ fontSize: 11 }}>24h 请求趋势<Hint title="过去 24h 每小时请求和错误数" /></span>} size="small" styles={{ body: { padding: 4 } }}>
                  <ResponsiveContainer width="100%" height={130}><AreaChart data={d} margin={{ left: -10, top: 2, bottom: 0 }}><CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="label" tick={{ fontSize: 9 }} interval={3} /><YAxis tick={{ fontSize: 9 }} /><RTooltip /><Area type="monotone" dataKey="count" name="请求" stroke={C.accent} fill={C.accent} fillOpacity={0.15} /><Area type="monotone" dataKey="errors" name="错误" stroke={C.error} fill={C.error} fillOpacity={0.15} /></AreaChart></ResponsiveContainer>
                </Card></Col>
                <Col xs={24} lg={12}><Card title={<span style={{ fontSize: 11 }}>24h Token 消耗<Hint title="过去 24h 每小时 Token 消耗" /></span>} size="small" styles={{ body: { padding: 4 } }}>
                  <ResponsiveContainer width="100%" height={130}><AreaChart data={d} margin={{ left: -10, top: 2, bottom: 0 }}><CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="label" tick={{ fontSize: 9 }} interval={3} /><YAxis tick={{ fontSize: 9 }} /><RTooltip /><Area type="monotone" dataKey="input_tokens" name="输入" stroke={C.accent} fill={C.accent} fillOpacity={0.15} /><Area type="monotone" dataKey="output_tokens" name="输出" stroke="#73d13d" fill="#73d13d" fillOpacity={0.15} /></AreaChart></ResponsiveContainer>
                </Card></Col>
              </Row>
            );
          })()}

          {(stats.by_model?.length > 0 || epData.length > 0) && (
            <Row gutter={[6, 6]} style={{ marginBottom: 8 }}>
              {stats.by_model?.length > 0 && <Col xs={24} lg={14}><Card title={<span style={{ fontSize: 11 }}><FireOutlined /> 模型分布</span>} size="small" styles={{ body: { padding: 4 } }}>
                <ResponsiveContainer width="100%" height={130}><BarChart data={stats.by_model} layout="vertical" margin={{ left: 10 }}><CartesianGrid strokeDasharray="3 3" /><XAxis type="number" tick={{ fontSize: 9 }} /><YAxis dataKey="model" type="category" width={80} tick={{ fontSize: 9 }} /><RTooltip /><Bar dataKey="count" name="请求" fill={C.accent} radius={[0, 4, 4, 0]} /></BarChart></ResponsiveContainer>
              </Card></Col>}
              {epData.length > 0 && <Col xs={24} lg={10}><Card title={<span style={{ fontSize: 11 }}><GlobalOutlined /> 端点分布</span>} size="small" styles={{ body: { padding: 4 } }}>
                <ResponsiveContainer width="100%" height={130}><PieChart><Pie data={epData} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={50} label={({ percent }: any) => `${((percent || 0) * 100).toFixed(0)}%`}>{epData.map((_, i) => <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />)}</Pie><RTooltip /></PieChart></ResponsiveContainer>
              </Card></Col>}
            </Row>
          )}
        </>
      )}

      {/* ═══ Request logs ═══ */}
      <Card title={<div style={{ display: 'flex', alignItems: 'center', gap: 4 }}><ClockCircleOutlined style={{ fontSize: 12 }} /><span style={{ fontSize: 12 }}>请求日志<Hint title="最近 500 条请求记录" /></span><Tag style={{ fontSize: 9, lineHeight: '14px' }}>{logs.length}</Tag></div>} size="small" extra={<Segmented size="small" value={logFilter} onChange={v => setLogFilter(v as string)} options={[{ label: '全部', value: 'all' }, { label: '流式', value: 'stream' }, { label: '错误', value: 'errors' }]} />}>
        <Table dataSource={fLogs} columns={logCols} rowKey="id" size="small" pagination={{ pageSize: 15, showSizeChanger: false, showTotal: t => `共 ${t} 条` }} scroll={{ x: 700 }} locale={{ emptyText: '暂无请求记录' }}
          expandable={{ expandedRowRender: r => (
            <div style={{ padding: '4px 0 4px 8px' }}>
              {r.status_code >= 400 && <div style={{ marginBottom: 6, padding: '6px 8px', border: '1px solid #ffccc7', borderRadius: 2, background: '#fff2f0' }}>
                <Typography.Text strong style={{ color: '#cf1322', fontSize: 11 }}>错误</Typography.Text>
                <pre style={{ margin: '4px 0 0', whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontSize: 10, color: '#cf1322', fontFamily: 'monospace' }}>{r.error_message || `HTTP ${r.status_code}`}</pre>
              </div>}
              <div style={{ display: 'grid', gridTemplateColumns: '80px minmax(0,1fr)', gap: '2px 8px', fontSize: 10 }}>
                <Typography.Text type="secondary">ID</Typography.Text><Typography.Text code style={{ fontSize: 10 }}>{r.id}</Typography.Text>
                <Typography.Text type="secondary">时间</Typography.Text><Typography.Text style={{ fontSize: 10 }}>{ft(r.created_at)}</Typography.Text>
                <Typography.Text type="secondary">端点</Typography.Text><Typography.Text code style={{ fontSize: 10 }}>{r.endpoint}</Typography.Text>
                <Typography.Text type="secondary">模型</Typography.Text><Typography.Text style={{ fontSize: 10 }}>{r.model || '-'}</Typography.Text>
                <Typography.Text type="secondary">流式</Typography.Text><Typography.Text style={{ fontSize: 10 }}>{r.stream ? '是' : '否'}</Typography.Text>
                <Typography.Text type="secondary">状态</Typography.Text><Typography.Text style={{ fontSize: 10 }}>{r.status_code}</Typography.Text>
                <Typography.Text type="secondary">Token</Typography.Text><Typography.Text style={{ fontSize: 10 }}>{r.input_tokens || 0}/{r.output_tokens || 0}</Typography.Text>
                <Typography.Text type="secondary">延迟</Typography.Text><Typography.Text style={{ fontSize: 10 }}>{flat(r.latency_ms)}</Typography.Text>
              </div>
            </div>
          )}} />
      </Card>
    </div>
  );
};

export default AccountDetail;