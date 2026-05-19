import React, { useEffect, useState } from 'react';
import ReactDOM from 'react-dom/client';
import { Alert, Button, Card, Col, ConfigProvider, Form, Input, Layout, List, Row, Select, Space, Statistic, Tag, Typography, message } from 'antd';
import 'antd/dist/reset.css';
import './styles.css';

type Overview = { mode: string; clusters: number; namespaces: number; workloads: number; alerts: number; environment: string };
type Cluster = { name: string; status: string; version: string; nodes: number };
type Workload = { name: string; namespace: string; kind: string; ready: string; image: string };
type AuditEvent = { id: string; at: string; actor: string; action: string; target: string; approved: boolean; allowed: boolean; reason: string };
type Tool = { name: string; description: string; write: boolean; requiresApproval: boolean };

const API_BASE = import.meta.env.VITE_API_BASE ?? '';
async function getJSON<T>(path: string): Promise<T> { const r = await fetch(`${API_BASE}${path}`); if (!r.ok) throw new Error(await r.text()); return r.json(); }

function App() {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [workloads, setWorkloads] = useState<Workload[]>([]);
  const [tools, setTools] = useState<Tool[]>([]);
  const [audit, setAudit] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [form] = Form.useForm();

  async function refresh() {
    setLoading(true);
    try {
      const [o, c, w, t, a] = await Promise.all([getJSON<Overview>('/api/v1/overview'), getJSON<Cluster[]>('/api/v1/clusters'), getJSON<Workload[]>('/api/v1/workloads'), getJSON<Tool[]>('/api/v1/tools'), getJSON<AuditEvent[]>('/api/v1/audit')]);
      setOverview(o); setClusters(c); setWorkloads(w); setTools(t); setAudit(a);
    } catch (err) { message.error(`API error: ${err instanceof Error ? err.message : String(err)}`); } finally { setLoading(false); }
  }
  useEffect(() => { void refresh(); }, []);

  async function execute(values: { tool: string; target: string; actor: string; approved?: boolean }) {
    const res = await fetch(`${API_BASE}/api/v1/tools/execute`, { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({...values, approved: Boolean(values.approved), parameters: {source: 'frontend'}}) });
    const body = await res.json();
    if (!res.ok) message.warning(body.error ?? 'tool rejected'); else message.success(body.message);
    await refresh();
  }

  return <ConfigProvider theme={{ token: { colorPrimary: '#1677ff' } }}>
    <Layout className="app"><Layout.Header className="header"><Typography.Title level={3}>ops-mcp</Typography.Title><Tag color="blue">mock-safe ops console</Tag></Layout.Header>
    <Layout.Content className="content"><Alert type="info" showIcon message="Mock mode is enabled by default. No real Kubernetes, Prometheus, shell, namespace deletion, PVC deletion, or kubectl exec actions are implemented." />
    <Row gutter={[16,16]} className="stats">
      <Col xs={12} md={6}><Card loading={loading}><Statistic title="Mode" value={overview?.mode ?? '-'} /></Card></Col>
      <Col xs={12} md={6}><Card loading={loading}><Statistic title="Clusters" value={overview?.clusters ?? 0} /></Card></Col>
      <Col xs={12} md={6}><Card loading={loading}><Statistic title="Workloads" value={overview?.workloads ?? 0} /></Card></Col>
      <Col xs={12} md={6}><Card loading={loading}><Statistic title="Alerts" value={overview?.alerts ?? 0} /></Card></Col>
    </Row>
    <Row gutter={[16,16]}>
      <Col xs={24} lg={12}><Card title="Clusters"><List dataSource={clusters} renderItem={(c) => <List.Item><Space><b>{c.name}</b><Tag color="green">{c.status}</Tag><span>{c.version}</span><span>{c.nodes} nodes</span></Space></List.Item>} /></Card></Col>
      <Col xs={24} lg={12}><Card title="Workloads"><List dataSource={workloads} renderItem={(w) => <List.Item><Space direction="vertical"><b>{w.namespace}/{w.name}</b><Space><Tag>{w.kind}</Tag><Tag color={w.ready.startsWith('3/') || w.ready.startsWith('1/') ? 'green':'orange'}>{w.ready}</Tag><code>{w.image}</code></Space></Space></List.Item>} /></Card></Col>
    </Row>
    <Row gutter={[16,16]} className="section"><Col xs={24} lg={10}><Card title="Safe tool execution">
      <Form form={form} layout="vertical" onFinish={execute} initialValues={{actor:'local-user', tool:'get_cluster_health', target:'local-dev'}}>
        <Form.Item name="actor" label="Actor" rules={[{required:true}]}><Input /></Form.Item>
        <Form.Item name="tool" label="Tool" rules={[{required:true}]}><Select options={tools.map(t => ({ value:t.name, label:`${t.name}${t.requiresApproval ? ' (approval)' : ''}` }))} /></Form.Item>
        <Form.Item name="target" label="Target" rules={[{required:true}]}><Input /></Form.Item>
        <Form.Item name="approved" label="Approval" valuePropName="checked"><Select options={[{value:false,label:'No approval'},{value:true,label:'Approved'}]} /></Form.Item>
        <Button type="primary" htmlType="submit">Execute audited mock tool</Button>
      </Form></Card></Col><Col xs={24} lg={14}><Card title="Audit events"><List dataSource={audit} renderItem={(e) => <List.Item><Space direction="vertical"><Space><Tag color={e.allowed?'green':'red'}>{e.allowed?'allowed':'blocked'}</Tag><b>{e.action}</b><span>{e.target}</span></Space><small>{e.id} · {e.actor} · {e.reason}</small></Space></List.Item>} /></Card></Col></Row>
    </Layout.Content></Layout></ConfigProvider>;
}

ReactDOM.createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>);
