import React, { useEffect, useState } from 'react';
import {
  Table, Button, Space, Modal, Form, Input, Switch, Select,
  message, Popconfirm, Tag, Typography, Tooltip, Card, Divider, Collapse, Row, Col,
} from 'antd';
import {
  PlusOutlined, DeleteOutlined, StarOutlined,
  SafetyCertificateOutlined, ReloadOutlined,
  QuestionCircleOutlined, EditOutlined,
  ExportOutlined, UploadOutlined, LoginOutlined, KeyOutlined,
  FlagOutlined, CodeOutlined, GlobalOutlined,
} from '@ant-design/icons';
import QRLoginModal from '../components/QRLoginModal';
import { useNavigate } from 'react-router-dom';
import { api, accountDisplayName } from '../api';
import type { Account } from '../api';

const BUILTIN_MODELS = [
  { label: 'deepseek-v4-flash（推荐）', value: 'deepseek-v4-flash' },
  { label: 'deepseek-chat', value: 'deepseek-chat' },
  { label: 'Qwen-QwQ-32B', value: 'Qwen-QwQ-32B' },
];

const maskUserId = (id: string): string => {
  if (!id) return '-';
  if (id.length <= 3) return id[0] + '***';
  return id.slice(0, 2) + '***' + id.slice(-2);
};

const fmtTokens = (n: number): string => {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return String(n);
};

const Accounts: React.FC = () => {
  const navigate = useNavigate();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [validating, setValidating] = useState<string | null>(null);
  const [autoLogging, setAutoLogging] = useState(false);
  const [qrModalOpen, setQrModalOpen] = useState(false);
  const [renameModalOpen, setRenameModalOpen] = useState(false);
  const [renameTarget, setRenameTarget] = useState<string>('');
  const [renameForm] = Form.useForm();
  const fileInputRef = React.useRef<HTMLInputElement>(null);
  const [importing, setImporting] = useState(false);

  const fetchAccounts = async () => {
    setLoading(true);
    try {
      const data = await api.listAccounts();
      setAccounts(data);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '获取账号列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchAccounts(); }, []);

  const handleAdd = async (values: { pt_key: string; user_id: string; is_default?: boolean; default_model?: string }) => {
    try {
      await api.addAccount(values);
      message.success(`账号「${values.user_id}」添加成功`);
      setModalOpen(false);
      form.resetFields();
      fetchAccounts();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '添加账号失败');
    }
  };

  const handleAutoLogin = async () => {
    setAutoLogging(true);
    try {
      const result = await api.autoLogin();
      message.success(`一键登录成功！账号「${result.nickname || result.user_id}」已添加`);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '一键登录失败');
    } finally {
      setAutoLogging(false);
    }
  };

  const handleRemove = async (userId: string, displayName: string) => {
    try {
      await api.removeAccount(userId);
      message.success(`账号「${displayName}」已删除`);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '删除账号失败');
    }
  };

  const handleSetDefault = async (userId: string, displayName: string) => {
    try {
      await api.setDefault(userId);
      message.success(`已将「${displayName}」设为默认账号`);
      fetchAccounts();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '设置默认账号失败');
    }
  };

  const handleRenewToken = async (userId: string) => {
    try {
      await api.renewToken(userId);
      message.success('API Token 已更新');
      fetchAccounts();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '更新 Token 失败');
    }
  };

  const handleValidate = async (userId: string, displayName: string) => {
    setValidating(userId);
    try {
      const result = await api.validateAccount(userId);
      if (result.valid) {
        message.success(`账号「${displayName}」验证通过，凭证有效`);
      } else {
        message.error(`账号「${displayName}」验证失败，凭证无效或已过期`);
      }
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '验证请求失败');
    } finally {
      setValidating(null);
    }
  };

  const handleRename = async (values: { new_name: string }) => {
    try {
      await api.updateRemark(renameTarget, values.new_name);
      message.success(`账号备注已更新为「${values.new_name}」`);
      setRenameModalOpen(false);
      renameForm.resetFields();
      fetchAccounts();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '更新备注失败');
    }
  };

  // Navigate to detail page only when clicking on non-action areas
  const goToDetail = (userId: string) => {
    navigate(`/accounts/${encodeURIComponent(userId)}`);
  };

  const columns = [
    {
      title: '账户名',
      dataIndex: 'user_id',
      key: 'user_id',
      width: 180,
      ellipsis: true,
      render: (_: unknown, record: Account) => (
        <Tooltip title={accountDisplayName(record)} placement="topLeft">
          <Typography.Text strong style={{ fontSize: 13 }}>{accountDisplayName(record)}</Typography.Text>
        </Tooltip>
      ),
    },
    {
      title: '用户 ID',
      dataIndex: 'user_id',
      key: 'user_id_uid',
      width: 100,
      render: (text: string) => (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>{maskUserId(text)}</Typography.Text>
      ),
    },
    {
      title: '活跃',
      dataIndex: 'active_sessions',
      key: 'active_sessions',
      width: 60,
      render: (val: number) => val > 0 ? <Tag color="blue" style={{ fontSize: 10, lineHeight: '16px' }}>{val}</Tag> : <Typography.Text type="secondary" style={{ fontSize: 12 }}>-</Typography.Text>,
    },
    {
      title: '今日请求',
      dataIndex: 'today_requests',
      key: 'today_requests',
      width: 85,
      render: (val: number, record: Account) => (
        <div style={{ lineHeight: 1.3 }}>
          <Typography.Text strong style={{ fontSize: 12 }}>{val}</Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 10, marginLeft: 4 }}>/ {record.total_requests}</Typography.Text>
        </div>
      ),
    },
    {
      title: 'Token',
      dataIndex: 'today_tokens',
      key: 'today_tokens',
      width: 85,
      render: (val: number, record: Account) => (
        <div style={{ lineHeight: 1.3 }}>
          <Typography.Text strong style={{ fontSize: 12 }}>{fmtTokens(val)}</Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 10, marginLeft: 4 }}>/ {fmtTokens(record.total_tokens)}</Typography.Text>
        </div>
      ),
    },
    {
      title: '凭证',
      key: 'credential_status',
      width: 70,
      render: (_: unknown, record: Account) => {
        const cv = record.credential_valid;
        if (cv === 1) return <Tag color="success" style={{ fontSize: 10, lineHeight: '16px' }}>有效</Tag>;
        if (cv === 0) return <Tag color="error" style={{ fontSize: 10, lineHeight: '16px' }}>已过期</Tag>;
        return <Tag color="processing" style={{ fontSize: 10, lineHeight: '16px' }}>检测中</Tag>;
      },
    },
    {
      title: '默认',
      dataIndex: 'is_default',
      key: 'is_default',
      width: 55,
      render: (val: boolean) => val ? <Tag color="blue" style={{ fontSize: 10, lineHeight: '16px' }}>默认</Tag> : null,
    },
    {
      title: '模型',
      dataIndex: 'default_model',
      key: 'default_model',
      width: 120,
      ellipsis: true,
      render: (val: string) => val ? <Tag color="green" style={{ fontSize: 10, lineHeight: '16px' }}>{val}</Tag> : <Typography.Text type="secondary" style={{ fontSize: 11 }}>未设置</Typography.Text>,
    },
    {
      title: '操作',
      key: 'actions',
      width: 340,
      onCell: () => ({ onClick: (e: React.MouseEvent) => e.stopPropagation(), style: { cursor: 'default' } }),
      render: (_: unknown, record: Account) => (
        <Space size={4} onClick={(e) => e.stopPropagation()}>
          <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} onClick={(e) => {
            e.stopPropagation();
            setRenameTarget(record.user_id);
            renameForm.setFieldsValue({ new_name: record.remark || accountDisplayName(record) });
            setRenameModalOpen(true);
          }}>
            <EditOutlined /> 修改备注
          </Button>
          {!record.is_default && (
            <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} onClick={(e) => { e.stopPropagation(); handleSetDefault(record.user_id, accountDisplayName(record)); }}>
              <StarOutlined /> 设为默认
            </Button>
          )}
          <Popconfirm
            title="重置 API Token？"
            description="重置后旧 Token 将立即失效"
            onConfirm={() => handleRenewToken(record.user_id)}
            onCancel={(e) => { if (e) e.stopPropagation(); }}
          >
            <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} onClick={(e) => e.stopPropagation()}>重置 Token</Button>
          </Popconfirm>
          <Button
            size="small"
            style={{ fontSize: 11, lineHeight: '18px' }}
            onClick={(e) => { e.stopPropagation(); handleValidate(record.user_id, accountDisplayName(record)); }}
            loading={validating === record.user_id}
          >
            验证凭据
          </Button>
          <Popconfirm
            title={`删除「${accountDisplayName(record)}」？`}
            description="客户端将无法访问"
            onConfirm={() => handleRemove(record.user_id, accountDisplayName(record))}
            onCancel={(e) => { if (e) e.stopPropagation(); }}
          >
            <Button size="small" danger style={{ fontSize: 11, lineHeight: '18px' }} onClick={(e) => e.stopPropagation()}><DeleteOutlined /> 删除账号</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const hasAnyUsableAccount = accounts.length > 0 && accounts.some(a => a.credential_valid !== 0);

  const OnboardingCard = () => (
    <Card style={{ marginBottom: 24, borderRadius: 2, border: '1px solid #e8e8e8', overflow: 'hidden' }} styles={{ body: { padding: 0 } }}>
      <div style={{ background: 'linear-gradient(135deg, #e8a050 0%, #c87618 100%)', padding: '32px 40px', textAlign: 'center' }}>
        <FlagOutlined style={{ fontSize: 36, color: '#fff', marginBottom: 12 }} />
        <Typography.Title level={3} style={{ color: '#fff', margin: '0 0 8px' }}>欢迎使用 AtomCode 2API</Typography.Title>
        <Typography.Text style={{ color: 'rgba(255,255,255,0.85)', fontSize: 15 }}>将 AtomCode Daemon 的 AI 能力转为标准的 OpenAI/Anthropic API</Typography.Text>
      </div>
      <div style={{ padding: '24px 32px' }}>
        <Row gutter={[32, 24]}>
          {[{ n: '1', t: '添加账号', d: '通过本机自动导入或手动添加 AtomCode 账号' }, { n: '2', t: '配置客户端', d: '使用 API Token 连接 Claude Code / Codex / 任意 OpenAI 客户端' }, { n: '3', t: '开始使用', d: '发送 API 请求，在 Dashboard 上查看使用统计' }].map(s => (
            <Col xs={24} md={8} key={s.n}>
              <div style={{ textAlign: 'center' }}>
                <div style={{ width: 48, height: 48, borderRadius: '50%', background: 'rgba(232,160,80,0.12)', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 12px', fontSize: 20, fontWeight: 700, color: '#e8a050' }}>{s.n}</div>
                <Typography.Text strong style={{ display: 'block', marginBottom: 4, fontSize: 15 }}>{s.t}</Typography.Text>
                <Typography.Text type="secondary" style={{ fontSize: 13 }}>{s.d}</Typography.Text>
              </div>
            </Col>
          ))}
        </Row>
      </div>
      <Divider style={{ margin: 0 }} />
      <div style={{ padding: '20px 32px', background: '#fafafa', textAlign: 'center' }}>
        <Typography.Title level={5} style={{ marginBottom: 16 }}>选择一种方式开始</Typography.Title>
        <Space wrap size="large">
          <Card hoverable style={{ width: 200, borderRadius: 2, textAlign: 'center' }} onClick={handleAutoLogin}>
            <LoginOutlined style={{ fontSize: 28, color: '#e8a050', marginBottom: 8 }} />
            <div style={{ fontWeight: 600 }}>一键导入</div>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>从本机 AtomCode 自动导入账号</Typography.Text>
          </Card>
          <Card hoverable style={{ width: 200, borderRadius: 2, textAlign: 'center' }} onClick={() => setModalOpen(true)}>
            <KeyOutlined style={{ fontSize: 28, color: '#faad14', marginBottom: 8 }} />
            <div style={{ fontWeight: 600 }}>手动添加</div>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>输入 ptKey 和用户 ID</Typography.Text>
          </Card>
          <Card hoverable style={{ width: 200, borderRadius: 2, textAlign: 'center' }} onClick={() => fileInputRef.current?.click()}>
            <UploadOutlined style={{ fontSize: 28, color: '#722ed1', marginBottom: 8 }} />
            <div style={{ fontWeight: 600 }}>批量导入</div>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>从 JSON 文件导入多个账号</Typography.Text>
          </Card>
          <Card hoverable style={{ width: 200, borderRadius: 2, textAlign: 'center' }} onClick={() => window.open('https://atomcode.atomgit.com/', '_blank')}>
            <GlobalOutlined style={{ fontSize: 28, color: '#e8a050', marginBottom: 8 }} />
            <div style={{ fontWeight: 600 }}>注册 AtomCode</div>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>前往官网注册账号</Typography.Text>
          </Card>
        </Space>
        <Divider style={{ margin: '20px 0 12px' }} />
        <Collapse ghost style={{ textAlign: 'left' }} items={[{
          key: 'client-setup', label: <span><CodeOutlined style={{ marginRight: 6 }} />已有账号？查看客户端配置方式</span>,
          children: (
            <div>
              <Typography.Text strong style={{ display: 'block', marginBottom: 8 }}>Claude Code</Typography.Text>
              <Typography.Paragraph code copyable style={{ background: '#f5f5f5', padding: '8px 12px', borderRadius: 3, marginBottom: 12, fontSize: 13 }}>{`export ANTHROPIC_BASE_URL=http://192.168.1.90:13457\nexport ANTHROPIC_API_KEY=sk-atmc-...`}</Typography.Paragraph>
              <Typography.Text strong style={{ display: 'block', marginBottom: 8 }}>OpenAI SDK</Typography.Text>
              <Typography.Paragraph code copyable style={{ background: '#f5f5f5', padding: '8px 12px', borderRadius: 3, marginBottom: 0, fontSize: 13 }}>{`from openai import OpenAI\nclient = OpenAI(base_url="http://192.168.1.90:13457/v1", api_key="sk-atmc-...")`}</Typography.Paragraph>
            </div>
          ),
        }]} />
      </div>
    </Card>
  );

  return (
    <div style={{ padding: '0 4px' }}>
      {(!accounts.length || !hasAnyUsableAccount) && <OnboardingCard />}

      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 8 }}>
        <Typography.Title level={5} style={{ margin: 0, fontSize: 15 }}>账号管理 <Typography.Text type="secondary" style={{ fontSize: 12, fontWeight: 400 }}>{accounts.length} 个</Typography.Text></Typography.Title>
        <Space wrap size={4}>
          <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} onClick={fetchAccounts} icon={<ReloadOutlined />}>刷新列表</Button>
          <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} onClick={handleAutoLogin} loading={autoLogging} icon={<LoginOutlined />}>一键导入账号</Button>
          <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} onClick={async () => {
            try {
              const result = await api.browserLogin();
              window.open(result.url, '_blank');
              message.loading({ content: '等待授权...', key: 'oauth', duration: 0 });
              for (let i = 0; i < Math.min(result.expires_in_seconds || 300, 600) / 2; i++) {
                await new Promise(r => setTimeout(r, 2000));
                try {
                  const poll = await api.oauthPoll(result.login_id);
                  if (poll.status === 'confirmed' || poll.ok) { message.success({ content: '授权成功！账号已导入', key: 'oauth' }); fetchAccounts(); return; }
                  if (poll.status === 'error' || poll.status === 'failed') { message.error({ content: poll.message || '授权失败', key: 'oauth' }); return; }
                } catch {}
              }
              message.warning({ content: '授权超时', key: 'oauth' });
            } catch (e: unknown) { message.error(e instanceof Error ? e.message : '获取链接失败'); }
          }} icon={<SafetyCertificateOutlined />}>OAuth 授权登录</Button>
          <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} icon={<GlobalOutlined />} onClick={() => window.open('https://atomcode.atomgit.com/', '_blank')}>注册新账号</Button>
          <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} onClick={async () => {
            try {
              const result = await api.exportAccounts();
              if (!result.accounts?.length) { message.warning('没有可导出的账号'); return; }
              const blob = new Blob([JSON.stringify(result.accounts, null, 2)], { type: 'application/json' });
              const url = URL.createObjectURL(blob); const a = document.createElement('a');
              a.href = url; a.download = `atomcode-accounts-${new Date().toISOString().slice(0, 10)}.json`; a.click();
              URL.revokeObjectURL(url); message.success(`已导出 ${result.count} 个`);
            } catch (e: unknown) { message.error(e instanceof Error ? e.message : '导出失败'); }
          }} icon={<ExportOutlined />}>导出账号</Button>
          <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} onClick={() => fileInputRef.current?.click()} icon={<UploadOutlined />} loading={importing}>导入账号</Button>
          <input ref={fileInputRef} type="file" accept=".json" style={{ display: 'none' }} onChange={async (e) => {
            const file = e.target.files?.[0]; if (!file) return;
            setImporting(true);
            try {
              const text = await file.text(); const accounts = JSON.parse(text);
              if (!Array.isArray(accounts) || !accounts.length) { message.error('文件格式错误'); return; }
              const result = await api.importAccounts(accounts);
              message.success(`导入完成：新增 ${result.added} 个，更新 ${result.updated} 个`); fetchAccounts();
            } catch (err: unknown) { message.error(err instanceof Error ? err.message : '导入失败'); } finally { setImporting(false); e.target.value = ''; }
          }} />
          <Button size="small" style={{ fontSize: 11, lineHeight: '18px' }} onClick={() => setModalOpen(true)} icon={<PlusOutlined />}>手动添加账号</Button>
        </Space>
      </div>

      <Table
        dataSource={accounts}
        columns={columns}
        rowKey="user_id"
        loading={loading}
        pagination={false}
        size="small"
        locale={{ emptyText: '暂无账号，请点击「一键导入」或「OAuth授权登录」按钮配置您的第一个账号' }}
        onRow={(record) => ({
          onClick: (e: React.MouseEvent) => {
            // Only navigate when clicking on non-action cells
            const target = e.target as HTMLElement;
            if (target.closest('.ant-table-cell') && !target.closest('[data-cell-action="true"]')) {
              goToDetail(record.user_id);
            }
          },
          style: { cursor: 'pointer' },
        })}
      />

      <Modal title="手动添加账号" open={modalOpen} onCancel={() => { setModalOpen(false); form.resetFields(); }} onOk={() => form.submit()} okText="添加" cancelText="取消" width={560}>
        <Form form={form} layout="vertical" onFinish={handleAdd}>
          <Form.Item name="pt_key" label={<Space size={4}>ptKey 凭证<Tooltip title="网页 OAuth 登录得到的 ptKey"><QuestionCircleOutlined style={{ color: '#999' }} /></Tooltip></Space>} rules={[{ required: true, message: '请输入 ptKey' }]}>
            <Input.Password placeholder="粘贴 ptKey" />
          </Form.Item>
          <Form.Item name="user_id" label={<Space size={4}>用户 ID<Tooltip title="与 ptKey 对应的用户 ID"><QuestionCircleOutlined style={{ color: '#999' }} /></Tooltip></Space>} rules={[{ required: true, message: '请输入用户 ID' }]}>
            <Input placeholder="例如：user-12345" />
          </Form.Item>
          <Form.Item name="default_model" label="默认模型">
            <Select placeholder="留空使用系统默认模型" options={BUILTIN_MODELS} allowClear />
          </Form.Item>
          <Form.Item name="is_default" valuePropName="checked" label={<Space size={4}>设为默认账号<Tooltip title="当客户端未提供路由密钥时，请求将自动路由到此默认账号"><QuestionCircleOutlined style={{ color: '#999' }} /></Tooltip></Space>}>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="修改账号备注" open={renameModalOpen} onCancel={() => { setRenameModalOpen(false); renameForm.resetFields(); }} onOk={() => renameForm.submit()} okText="确认" cancelText="取消">
        <Form form={renameForm} layout="vertical" onFinish={handleRename}>
          <Form.Item name="new_name" label="备注名" rules={[{ required: true, message: '请输入备注名' }]}>
            <Input placeholder="输入备注名，例如：我的主账号" />
          </Form.Item>
        </Form>
      </Modal>

      <QRLoginModal open={qrModalOpen} onClose={() => setQrModalOpen(false)} onSuccess={fetchAccounts} onAutoLogin={handleAutoLogin} />
    </div>
  );
};

export default Accounts;