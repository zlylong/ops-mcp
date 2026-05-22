import React from 'react';
import { Button, Card, Empty, message, Popconfirm, Select, Space, Table, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../services/api';
import type { Approval } from '../types';
import { StatusTag, formatTime, shortId } from '../components/utils';

export function ApprovalsPage() {
  const approvals = useQuery({ queryKey: ['approvals'], queryFn: api.approvals, refetchInterval: 5000 });
  const [status, setStatus] = React.useState('pending');
  const queryClient = useQueryClient();
  const decide = useMutation({
    mutationFn: ({ id, action }: { id: string; action: 'approve' | 'reject' }) => action === 'approve' ? api.approve(id) : api.reject(id),
    onSuccess: () => { message.success('审批状态已更新，批准的任务会自动执行'); queryClient.invalidateQueries({ queryKey: ['approvals'] }); queryClient.invalidateQueries({ queryKey: ['executions'] }); queryClient.invalidateQueries({ queryKey: ['audit'] }); queryClient.invalidateQueries({ queryKey: ['summary'] }); },
    onError: (err) => message.error(err instanceof Error ? err.message : '审批失败'),
  });
  const data = (approvals.data ?? []).filter((item) => status === 'all' || item.status === status);
  const columns = [
    { title: 'ID', dataIndex: 'id', render: (v: string) => <Typography.Text code>{shortId(v)}</Typography.Text> },
    { title: '状态', dataIndex: 'status', render: (v: string) => <StatusTag status={v} /> },
    { title: '工具', dataIndex: 'tool', render: (v: string) => <Typography.Text code>{v}</Typography.Text> },
    { title: '执行人', dataIndex: 'actor' },
    { title: '目标', dataIndex: 'target' },
    { title: '原因', dataIndex: 'reason', ellipsis: true },
    { title: '创建时间', dataIndex: 'createdAt', render: formatTime },
    { title: '操作', render: (_: unknown, row: Approval) => row.status === 'pending' ? <Space><Popconfirm title="确认批准该请求？" onConfirm={() => decide.mutate({ id: row.id, action: 'approve' })}><Button type="primary" size="small">批准</Button></Popconfirm><Popconfirm title="确认拒绝该请求？" onConfirm={() => decide.mutate({ id: row.id, action: 'reject' })}><Button danger size="small">拒绝</Button></Popconfirm></Space> : '-' },
  ];
  return <div className="page"><div className="page-title"><div><Typography.Title level={2}>任务审批中心</Typography.Title><Typography.Text type="secondary">处理需要人工确认的中高风险工具执行请求。</Typography.Text></div></div><Card className="section"><Space wrap className="toolbar"><Select value={status} onChange={setStatus} style={{ width: 160 }} options={[{ value: 'pending', label: '待审批' }, { value: 'all', label: '全部' }, { value: 'approved', label: '已批准' }, { value: 'rejected', label: '已拒绝' }]} /></Space><Table rowKey="id" className="section" loading={approvals.isLoading || decide.isPending} columns={columns} dataSource={data} locale={{ emptyText: <Empty description="暂无审批请求" /> }} pagination={{ pageSize: 10 }} /></Card></div>;
};
