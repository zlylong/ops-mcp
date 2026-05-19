import React, { useEffect, useMemo, useState } from 'react';
import ReactDOM from 'react-dom/client';
import { Alert, Button, Card, Col, ConfigProvider, Empty, Form, Input, Layout, List, Modal, Row, Select, Space, Spin, Statistic, Tag, Typography, message } from 'antd';
import 'antd/dist/reset.css';
import './styles.css';

type Summary = { mode: string; environment: string; tools: number; executions: number; auditRecords: number; approvals: number };
type Risk = 'low' | 'medium' | 'high' | 'critical';
type Tool = { name: string; description: string; category: string; readOnly: boolean; risk: Risk; requiresApproval: boolean; inputSchema: Record<string, string> };
type AuditEvent = { id: string; executionId?: string; at: string; actor: string; role: string; action: string; target: string; allowed: boolean; reason: string; parameters?: Record<string, unknown> };
type Execution = { id: string; tool: string; actor: string; role: string; target: string; status: string; reason: string; createdAt: string };
type ExecuteValues = { tool: string; actor: string; role: string; target: string; approved: boolean; parametersJson?: string };

const API_BASE = import.meta.env.VITE_API_BASE ?? '';
async function getJSON<T>(path: string): Promise<T> { const r = await fetch(`${API_BASE}${path}`); if (!r.ok) throw new Error(await r.text()); return r.json(); }

function parseParameters(text?: string): Record<string, unknown> {
  if (!text?.trim()) return {};
  const parsed: unknown = JSON.parse(text);
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error('Parameters must be a JSON object');
  return parsed as Record<string, unknown>;
}

function App() {
  const [summary, setSummary] = useState<Summary | null>(null);
  const [tools, setTools] = useState<Tool[]>([]);
  const [audit, setAudit] = useState<AuditEvent[]>([]);
  const [executions, setExecutions] = useState<Execution[]>([]);
  const [loading, setLoading] = useState(true);
  const [form] = Form.useForm<ExecuteValues>();
  const selectedToolName = Form.useWatch('tool', form);
  const selectedTool = useMemo(() => tools.find((tool) => tool.name === selectedToolName), [tools, selectedToolName]);

  async function refresh() {
    setLoading(true);
    try {
      const [s, t, a, e] = await Promise.all([
        getJSON<Summary>('/api/v1/dashboard/summary'),
        getJSON<Tool[]>('/api/v1/tools'),
        getJSON<AuditEvent[]>('/api/v1/audit'),
        getJSON<Execution[]>('/api/v1/executions'),
      ]);
      setSummary(s); setTools(t); setAudit(a); setExecutions(e);
    } catch (err) { message.error(`API error: ${err instanceof Error ? err.message : String(err)}`); } finally { setLoading(false); }
  }
  useEffect(() => { void refresh(); }, []);

  async function submitExecution(values: ExecuteValues) {
    const run = async () => {
      let parameters: Record<string, unknown>;
      try { parameters = parseParameters(values.parametersJson); } catch (err) { message.error(err instanceof Error ? err.message : String(err)); return; }
      const res = await fetch(`${API_BASE}/api/v1/tools/${encodeURIComponent(values.tool)}/execute`, {
        method: 'POST', headers: {'Content-Type':'application/json'},
        body: JSON.stringify({actor: values.actor, role: values.role, target: values.target, approved: Boolean(values.approved), parameters}),
      });
      const body: { error?: string; message?: string } = await res.json();
      if (!res.ok) message.warning(body.error ?? 'tool rejected'); else message.success(body.message ?? 'tool executed');
      await refresh();
    };
    if (selectedTool?.risk === 'medium' || selectedTool?.risk === 'high' || selectedTool?.requiresApproval) {
      Modal.confirm({ title: 'Confirm higher-risk operation', content: `Execute ${values.tool}? This action is policy checked and audited.`, okText: 'Execute', cancelText: 'Cancel', onOk: run });
      return;
    }
    await run();
  }

  return <ConfigProvider theme={{ token: { colorPrimary: '#1677ff' } }}>
    <Layout className="app"><Layout.Header className="header"><Typography.Title level={3}>ops-mcp</Typography.Title><Tag color="blue">read-only mock Ops MCP</Tag></Layout.Header>
    <Layout.Content className="content"><Alert type="info" showIcon message="Mock mode is enabled by default. All tool calls go through registry, policy, audit, and execution history. No shell, kubectl exec, or resource deletion tools are implemented." />
    <Row gutter={[16,16]} className="stats">
      <Col xs={12} md={6}><Card loading={loading}><Statistic title="Mode" value={summary?.mode ?? '-'} /></Card></Col>
      <Col xs={12} md={6}><Card loading={loading}><Statistic title="Tools" value={summary?.tools ?? 0} /></Card></Col>
      <Col xs={12} md={6}><Card loading={loading}><Statistic title="Executions" value={summary?.executions ?? 0} /></Card></Col>
      <Col xs={12} md={6}><Card loading={loading}><Statistic title="Approvals" value={summary?.approvals ?? 0} /></Card></Col>
    </Row>
    <Row gutter={[16,16]} className="section">
      <Col xs={24} lg={12}><Card title="Tool Registry">{loading ? <Spin /> : tools.length === 0 ? <Empty /> : <List dataSource={tools} renderItem={(tool) => <List.Item><Space direction="vertical"><Space><b>{tool.name}</b><Tag>{tool.category}</Tag><Tag color={tool.readOnly ? 'green' : 'orange'}>{tool.readOnly ? 'read-only' : 'write'}</Tag><Tag color={tool.risk === 'medium' ? 'gold' : 'blue'}>{tool.risk}</Tag></Space><span>{tool.description}</span></Space></List.Item>} />}</Card></Col>
      <Col xs={24} lg={12}><Card title="Execution History">{loading ? <Spin /> : executions.length === 0 ? <Empty description="No executions yet" /> : <List dataSource={executions} renderItem={(e) => <List.Item><Space direction="vertical"><Space><Tag color={e.status === 'succeeded' ? 'green' : 'red'}>{e.status}</Tag><b>{e.tool}</b><span>{e.target}</span></Space><small>{e.id} · {e.actor} · {e.reason}</small></Space></List.Item>} />}</Card></Col>
    </Row>
    <Row gutter={[16,16]} className="section"><Col xs={24} lg={10}><Card title="Audited tool execution">
      <Form form={form} layout="vertical" onFinish={submitExecution} initialValues={{actor:'local-user', role:'viewer', tool:'k8s.list_pods', target:'local-dev', approved:false, parametersJson:'{"namespace":"default"}'}}>
        <Form.Item name="actor" label="Actor" rules={[{required:true}]}><Input /></Form.Item>
        <Form.Item name="role" label="Role" rules={[{required:true}]}><Select options={[{value:'viewer',label:'viewer'},{value:'operator',label:'operator'},{value:'admin',label:'admin'}]} /></Form.Item>
        <Form.Item name="tool" label="Tool" rules={[{required:true}]}><Select options={tools.map(t => ({ value:t.name, label:`${t.name} (${t.risk})` }))} /></Form.Item>
        <Form.Item name="target" label="Target"><Input /></Form.Item>
        <Form.Item name="approved" label="Approval"><Select options={[{value:false,label:'No approval'},{value:true,label:'Approved'}]} /></Form.Item>
        <Form.Item name="parametersJson" label="Parameters JSON"><Input.TextArea rows={5} /></Form.Item>
        <Button type="primary" htmlType="submit">Execute through policy + audit</Button>
      </Form></Card></Col><Col xs={24} lg={14}><Card title="Audit Records">{loading ? <Spin /> : audit.length === 0 ? <Empty description="No audit records yet" /> : <List dataSource={audit} renderItem={(e) => <List.Item><Space direction="vertical"><Space><Tag color={e.allowed?'green':'red'}>{e.allowed?'allowed':'blocked'}</Tag><b>{e.action}</b><span>{e.target}</span></Space><small>{e.id} · {e.actor} · {e.role} · {e.reason}</small></Space></List.Item>} />}</Card></Col></Row>
    </Layout.Content></Layout></ConfigProvider>;
}

ReactDOM.createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>);
