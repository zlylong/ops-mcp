import { Row, Col, Card, Statistic, Typography, Table } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../services/api';
import type { Summary } from '../types';
import { StatusTag } from '../components/utils';

export function Dashboard() {
  const summary = useQuery({ queryKey: ['summary'], queryFn: api.summary });
  const executions = useQuery({ queryKey: ['executions'], queryFn: api.executions });
  const approvals = useQuery({ queryKey: ['approvals'], queryFn: api.approvals });
  const tools = useQuery({ queryKey: ['tools'], queryFn: api.tools });
  const failed = (executions.data ?? []).filter((item) => item.status !== 'succeeded').length;
  const today = (executions.data ?? []).filter((item) => new Date(item.createdAt).toDateString() === new Date().toDateString()).length;
  const pending = (approvals.data ?? []).filter((item) => item.status === 'pending').length;
  const riskCounts = (tools.data ?? []).reduce<Record<string, number>>((acc, tool) => ({ ...acc, [tool.risk]: (acc[tool.risk] ?? 0) + 1 }), {});
  const chart = { tooltip: { trigger: 'item' }, legend: { bottom: 0 }, series: [{ type: 'pie', radius: ['45%', '70%'], data: Object.entries(riskCounts).map(([name, value]) => ({ name, value })) }] };

  return (
    <div style={{ padding: 16 }}>
      <Typography.Title level={2}>仪表盘</Typography.Title>
      <Typography.Text type="secondary">用于审计 MCP 工具使用的运维指挥中心。</Typography.Text>
      <div style={{ marginTop: 16 }}>
        <Row gutter={[16, 16]}>
          <Col xs={12} lg={6}><Card><Statistic title="活跃告警" value={failed} prefix={<ExclamationCircleOutlined />} valueStyle={{ color: failed ? '#cf1322' : '#3f8600' }} /></Card></Col>
          <Col xs={12} lg={6}><Card><Statistic title="待审批" value={pending} /></Card></Col>
          <Col xs={12} lg={6}><Card><Statistic title="今日执行" value={today} /></Card></Col>
          <Col xs={12} lg={6}><Card><Statistic title="失败执行" value={failed} /></Card></Col>
        </Row>
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} lg={15}>
            <Card title="最近执行">
              <StatusTable data={(executions.data ?? []).slice(0, 8)} loading={executions.isLoading} compact />
            </Card>
          </Col>
          <Col xs={24} lg={9}>
            <Card title="风险分布图">
              <ReactECharts option={chart} style={{ height: 320 }} />
              <Typography.Text type="secondary">模式: {summary.data?.mode ?? '-'} · 环境: {summary.data?.environment ?? '-'}</Typography.Text>
            </Card>
          </Col>
        </Row>
      </div>
    </div>
  );
}

function StatusTable({ data, loading, compact }: { data: any[]; loading?: boolean; compact?: boolean }) {
  const columns: any = [
    { title: '状态', dataIndex: 'status', render: (s: string) => <StatusTag status={s} /> },
    { title: '工具', dataIndex: 'tool', render: (t: string) => <Typography.Text code>{t}</Typography.Text> },
    { title: '执行人', dataIndex: 'actor', responsive: ['md'] },
    { title: '目标', dataIndex: 'target', responsive: ['lg'] },
    { title: '策略原因', dataIndex: 'reason', ellipsis: true },
  ];
  return <Table rowKey="id" loading={loading} columns={compact ? columns : [...columns, { title: '创建时间', dataIndex: 'createdAt', render: (v: string) => v ? new Date(v).toLocaleString() : '-' }]} dataSource={data} pagination={compact ? false : { pageSize: 10 }} />;
}
