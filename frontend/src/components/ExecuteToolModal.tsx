import React from 'react';
import { Alert, Button, Form, Input, Modal, Space, Typography, message } from 'antd';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../services/api';
import type { ExecuteResult, Tool } from '../types';
import { JsonBlock, RiskTag, ReadOnlyTag, defaultInput, parseJsonObject } from './utils';

type Props = { tool?: Tool; open: boolean; onClose: () => void };
type FormValues = { target: string; parameters: string };

export function ExecuteToolModal({ tool, open, onClose }: Props) {
  const [form] = Form.useForm<FormValues>();
  const queryClient = useQueryClient();
  const [result, setResult] = React.useState<ExecuteResult | null>(null);

  React.useEffect(() => {
    if (open && tool) {
      form.setFieldsValue({ target: 'local-dev', parameters: defaultInput(tool) });
      setResult(null);
    }
  }, [open, tool, form]);

  const mutation = useMutation({
    mutationFn: async (values: FormValues) => {
      if (!tool) throw new Error('tool is required');
      return api.execute(tool.name, {
        target: values.target,
        parameters: parseJsonObject(values.parameters),
      });
    },
    onSuccess: (data) => {
      setResult(data);
      message.success(data.status === 'pending_approval' ? '已提交审批' : '执行完成');
      queryClient.invalidateQueries({ queryKey: ['summary'] });
      queryClient.invalidateQueries({ queryKey: ['executions'] });
      queryClient.invalidateQueries({ queryKey: ['audit'] });
      queryClient.invalidateQueries({ queryKey: ['approvals'] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : '执行失败'),
  });

  return (
    <Modal title={tool ? `执行工具：${tool.name}` : '执行工具'} open={open} onCancel={onClose} width={760} footer={null} destroyOnHidden>
      {tool && (
        <Space direction="vertical" size="middle" className="full">
          <Space wrap>
            <ReadOnlyTag readOnly={tool.readOnly} />
            <RiskTag risk={tool.risk} />
            {tool.requiresApproval || tool.risk !== 'low' ? <Alert type="warning" showIcon message="该工具可能需要管理员审批；执行人和角色由后端认证身份决定。" /> : null}
          </Space>
          <Typography.Paragraph type="secondary">{tool.description}</Typography.Paragraph>
          <Form layout="vertical" form={form} onFinish={(values) => mutation.mutate(values)}>
            <Space className="full" size="middle" wrap>
              <Form.Item name="target" label="目标" rules={[{ required: true }]}><Input placeholder="local-dev" /></Form.Item>
            </Space>
            <Form.Item name="parameters" label="参数 JSON" rules={[{ required: true }, { validator: async (_, value) => { parseJsonObject(value); } }]}>
              <Input.TextArea rows={8} spellCheck={false} className="json-input" />
            </Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={mutation.isPending}>执行</Button>
              <Button onClick={onClose}>关闭</Button>
            </Space>
          </Form>
          {result ? <Alert type={result.status === 'pending_approval' ? 'warning' : 'success'} showIcon message={result.message || result.status} description={<JsonBlock value={result} height={240} />} /> : null}
        </Space>
      )}
    </Modal>
  );
}
