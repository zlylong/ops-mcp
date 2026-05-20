import React from 'react';
import { Button, Card, Col, Drawer, Empty, Form, Input, Modal, Popconfirm, Row, Select, Space, Switch, Table, Typography, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { api } from '../services/api';
import type { Risk, Tool, ToolRequest } from '../types';
import { ExecuteToolModal } from '../components/ExecuteToolModal';
import { JsonBlock, ReadOnlyTag, RiskTag } from '../components/utils';

type ToolFormValues = Omit<ToolRequest, 'inputSchema'> & { inputSchemaText: string };
const defaultSchema = JSON.stringify({ namespace: 'string?' }, null, 2);
function toFormValues(tool?: Tool): ToolFormValues { return { name: tool?.name ?? '', description: tool?.description ?? '', category: tool?.category ?? 'custom', readOnly: tool?.readOnly ?? true, risk: tool?.risk ?? 'low', requiresApproval: tool?.requiresApproval ?? false, inputSchemaText: JSON.stringify(tool?.inputSchema ?? JSON.parse(defaultSchema), null, 2) }; }
function toRequest(values: ToolFormValues): ToolRequest { const inputSchema = JSON.parse(values.inputSchemaText || '{}') as Record<string, string>; return { name: values.name.trim(), description: values.description.trim(), category: values.category.trim(), readOnly: values.readOnly, risk: values.risk, requiresApproval: values.requiresApproval, inputSchema }; }

export function ToolsPage() {
  const queryClient = useQueryClient();
  const [messageApi, contextHolder] = message.useMessage();
  const tools = useQuery({ queryKey: ['tools'], queryFn: api.tools });
  const [q, setQ] = React.useState('');
  const [category, setCategory] = React.useState<string>('all');
  const [risk, setRisk] = React.useState<Risk | 'all'>('all');
  const [readOnly, setReadOnly] = React.useState<'all' | 'true' | 'false'>('all');
  const [detail, setDetail] = React.useState<Tool | null>(null);
  const [executeTool, setExecuteTool] = React.useState<Tool | undefined>();
  const [editingTool, setEditingTool] = React.useState<Tool | undefined>();
  const [formOpen, setFormOpen] = React.useState(false);
  const [form] = Form.useForm<ToolFormValues>();
  const invalidateTools = async () => { await queryClient.invalidateQueries({ queryKey: ['tools'] }); await queryClient.invalidateQueries({ queryKey: ['summary'] }); };
  const createMutation = useMutation({ mutationFn: api.createTool, onSuccess: async () => { await invalidateTools(); messageApi.success('工具已新增'); setFormOpen(false); } });
  const updateMutation = useMutation({ mutationFn: ({ name, data }: { name: string; data: ToolRequest }) => api.updateTool(name, data), onSuccess: async (updated) => { await invalidateTools(); messageApi.success('工具已更新'); setFormOpen(false); setDetail(updated); } });
  const deleteMutation = useMutation({ mutationFn: api.deleteTool, onSuccess: async (_, name) => { await invalidateTools(); messageApi.success('工具已删除'); if (detail?.name === name) setDetail(null); } });
  const categories = Array.from(new Set((tools.data ?? []).map((tool) => tool.category))).filter(Boolean);
  const data = (tools.data ?? []).filter((tool) => { const keyword = q.trim().toLowerCase(); return (!keyword || `${tool.name} ${tool.description} ${tool.category}`.toLowerCase().includes(keyword)) && (category === 'all' || tool.category === category) && (risk === 'all' || tool.risk === risk) && (readOnly === 'all' || String(tool.readOnly) === readOnly); });
  const openCreate = () => { setEditingTool(undefined); form.setFieldsValue(toFormValues()); setFormOpen(true); };
  const openEdit = (tool: Tool) => { setEditingTool(tool); form.setFieldsValue(toFormValues(tool)); setFormOpen(true); };
  const submitForm = async () => { const values = await form.validateFields(); let req: ToolRequest; try { req = toRequest(values); } catch { messageApi.error('输入 Schema 必须是合法 JSON 对象'); return; } if (editingTool) updateMutation.mutate({ name: editingTool.name, data: req }); else createMutation.mutate(req); };
  const columns = [
    { title: '工具', dataIndex: 'name', render: (_: string, row: Tool) => <Button type="link" onClick={() => setDetail(row)}>{row.name}</Button> },
    { title: '分类', dataIndex: 'category' },
    { title: '风险', dataIndex: 'risk', render: (v: string) => <RiskTag risk={v} /> },
    { title: '类型', dataIndex: 'readOnly', render: (v: boolean) => <ReadOnlyTag readOnly={v} /> },
    { title: '说明', dataIndex: 'description', ellipsis: true },
    { title: '操作', width: 260, render: (_: unknown, row: Tool) => <Space><Button onClick={() => setDetail(row)}>详情</Button><Button icon={<EditOutlined />} onClick={() => openEdit(row)}>编辑</Button><Button type="primary" onClick={() => setExecuteTool(row)}>执行</Button><Popconfirm title="删除工具" description={`确认删除 ${row.name}？`} okText="删除" okButtonProps={{ danger: true }} onConfirm={() => deleteMutation.mutate(row.name)}><Button danger icon={<DeleteOutlined />} /></Popconfirm></Space> },
  ];
  return <div className="page">{contextHolder}<div className="page-title"><div><Typography.Title level={2}>工具中心</Typography.Title><Typography.Text type="secondary">查看、增删改查 MCP 工具，按策略执行并自动审计。</Typography.Text></div><Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增工具</Button></div><Card className="section"><Space wrap className="toolbar"><Input.Search placeholder="搜索工具/说明" allowClear onSearch={setQ} onChange={(e) => setQ(e.target.value)} style={{ width: 260 }} /><Select value={category} onChange={setCategory} style={{ width: 160 }} options={[{ value: 'all', label: '全部分类' }, ...categories.map((value) => ({ value, label: value }))]} /><Select value={risk} onChange={setRisk} style={{ width: 140 }} options={[{ value: 'all', label: '全部风险' }, { value: 'low', label: '低' }, { value: 'medium', label: '中' }, { value: 'high', label: '高' }, { value: 'critical', label: '严重' }]} /><Select value={readOnly} onChange={setReadOnly} style={{ width: 140 }} options={[{ value: 'all', label: '全部类型' }, { value: 'true', label: '只读' }, { value: 'false', label: '变更' }]} /></Space><Table rowKey="name" className="section" loading={tools.isLoading} columns={columns} dataSource={data} locale={{ emptyText: <Empty description="暂无工具" /> }} pagination={{ pageSize: 8 }} /></Card><Drawer title={detail?.name} width={620} open={!!detail} onClose={() => setDetail(null)} extra={detail ? <Space><Button icon={<EditOutlined />} onClick={() => openEdit(detail)}>编辑</Button><Button type="primary" onClick={() => setExecuteTool(detail)}>执行</Button></Space> : null}>{detail ? <Space direction="vertical" className="full" size="middle"><Space wrap><RiskTag risk={detail.risk} /><ReadOnlyTag readOnly={detail.readOnly} /></Space><Typography.Paragraph>{detail.description}</Typography.Paragraph><Row gutter={12}><Col span={12}><Card size="small" title="分类">{detail.category}</Card></Col><Col span={12}><Card size="small" title="是否需要审批">{detail.requiresApproval ? '是' : '否'}</Card></Col></Row><Card size="small" title="输入 Schema"><JsonBlock value={detail.inputSchema} /></Card></Space> : null}</Drawer><Modal title={editingTool ? '编辑工具' : '新增工具'} open={formOpen} onOk={submitForm} confirmLoading={createMutation.isPending || updateMutation.isPending} onCancel={() => setFormOpen(false)} width={720} destroyOnHidden><Form form={form} layout="vertical" initialValues={toFormValues()}><Row gutter={12}><Col span={12}><Form.Item name="name" label="工具名称" rules={[{ required: true, message: '请输入工具名称' }, { pattern: /^[^\s/\\]+$/, message: '不能包含空格或斜杠' }]}><Input disabled={!!editingTool} placeholder="custom.echo" /></Form.Item></Col><Col span={12}><Form.Item name="category" label="分类" rules={[{ required: true, message: '请输入分类' }]}><Input placeholder="custom" /></Form.Item></Col></Row><Form.Item name="description" label="说明" rules={[{ required: true, message: '请输入说明' }]}><Input.TextArea rows={2} /></Form.Item><Row gutter={12}><Col span={8}><Form.Item name="risk" label="风险" rules={[{ required: true }]}><Select options={[{ value: 'low', label: '低' }, { value: 'medium', label: '中' }, { value: 'high', label: '高' }, { value: 'critical', label: '严重' }]} /></Form.Item></Col><Col span={8}><Form.Item name="readOnly" label="只读工具" valuePropName="checked"><Switch checkedChildren="只读" unCheckedChildren="变更" /></Form.Item></Col><Col span={8}><Form.Item name="requiresApproval" label="需要审批" valuePropName="checked"><Switch checkedChildren="需要" unCheckedChildren="无需" /></Form.Item></Col></Row><Form.Item name="inputSchemaText" label="输入 Schema(JSON 对象)" rules={[{ required: true, message: '请输入输入 Schema' }]}><Input.TextArea rows={8} spellCheck={false} /></Form.Item></Form></Modal><ExecuteToolModal tool={executeTool} open={!!executeTool} onClose={() => setExecuteTool(undefined)} /></div>;
}
