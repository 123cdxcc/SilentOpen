import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ProcessDetails from '../ProcessDetails.vue'
import type { ProcessInfo } from '../service'

const process = (overrides: Partial<ProcessInfo> = {}): ProcessInfo => ({
  pid: 100,
  ppid: 1,
  parentName: 'launchd',
  name: 'node',
  project: 'demo',
  cwd: '/demo',
  listeners: [{ ip: '127.0.0.1', port: 3000 }],
  command: 'node server',
  startedAt: 1,
  ...overrides,
})

describe('ProcessDetails', () => {
  it('renders parent, working directory and full command', () => {
    const wrapper = mount(ProcessDetails, { props: { process: process() } })
    expect(wrapper.text()).toContain('launchd')
    expect(wrapper.text()).toContain('PID 1')
    expect(wrapper.text()).toContain('/demo')
    expect(wrapper.text()).toContain('node server')
    expect(wrapper.text()).not.toContain('结束进程不可用')
  })

  it('explains why termination is unavailable', () => {
    const wrapper = mount(ProcessDetails, { props: { process: process({ pid: 1 }) } })
    expect(wrapper.text()).toContain('结束进程不可用')
    expect(wrapper.text()).toContain('系统保留进程不可结束')
  })

  it('marks unreadable fields as unknown', () => {
    const wrapper = mount(ProcessDetails, {
      props: { process: process({ parentName: '', ppid: undefined, cwd: '', command: '' }) },
    })
    expect(wrapper.text()).toContain('父进程')
    expect(wrapper.text()).toContain('PID 未知')
    expect(wrapper.text()).toContain('未知')
  })
})
