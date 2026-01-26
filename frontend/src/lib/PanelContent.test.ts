import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import PanelContent from './PanelContent.svelte';

describe('PanelContent', () => {
  afterEach(() => {
    cleanup();
  });

  describe('rendering by type', () => {
    it('renders chat placeholder when type is chat', () => {
      const { container } = render(PanelContent, { props: { type: 'chat', panelId: 'left' } });
      const header = container.querySelector('[data-testid="panel-header"]');
      expect(header?.textContent).toContain('Chat');
    });

    it('renders review placeholder when type is review', () => {
      const { container } = render(PanelContent, { props: { type: 'review', panelId: 'left' } });
      const header = container.querySelector('[data-testid="panel-header"]');
      expect(header?.textContent).toContain('Review');
    });

    it('renders terminal placeholder when type is terminal', () => {
      const { container } = render(PanelContent, { props: { type: 'terminal', panelId: 'right' } });
      const header = container.querySelector('[data-testid="panel-header"]');
      expect(header?.textContent).toContain('Terminal');
    });

    it('renders empty state when type is null', () => {
      const { container } = render(PanelContent, { props: { type: null, panelId: 'left' } });
      const empty = container.querySelector('[data-testid="panel-empty"]');
      expect(empty).toBeTruthy();
    });
  });

  describe('panel identification', () => {
    it('includes panelId in data attribute', () => {
      const { container } = render(PanelContent, { props: { type: 'chat', panelId: 'left' } });
      const panel = container.querySelector('[data-panel-id="left"]');
      expect(panel).toBeTruthy();
    });

    it('uses right panelId correctly', () => {
      const { container } = render(PanelContent, { props: { type: 'review', panelId: 'right' } });
      const panel = container.querySelector('[data-panel-id="right"]');
      expect(panel).toBeTruthy();
    });
  });

  describe('header with Cmd+K hint', () => {
    it('shows Cmd+K hint in header for chat', () => {
      const { container } = render(PanelContent, { props: { type: 'chat', panelId: 'left' } });
      const header = container.querySelector('[data-testid="panel-header"]');
      expect(header?.textContent).toContain('Cmd+K');
    });

    it('shows Cmd+K hint in header for review', () => {
      const { container } = render(PanelContent, { props: { type: 'review', panelId: 'right' } });
      const header = container.querySelector('[data-testid="panel-header"]');
      expect(header?.textContent).toContain('Cmd+K');
    });

    it('shows Cmd+K hint in header for terminal', () => {
      const { container } = render(PanelContent, { props: { type: 'terminal', panelId: 'left' } });
      const header = container.querySelector('[data-testid="panel-header"]');
      expect(header?.textContent).toContain('Cmd+K');
    });

    it('shows Cmd+K hint in empty state', () => {
      const { container } = render(PanelContent, { props: { type: null, panelId: 'left' } });
      const empty = container.querySelector('[data-testid="panel-empty"]');
      expect(empty?.textContent).toContain('Cmd+K');
    });
  });

  describe('content areas', () => {
    it('renders content area for chat type', () => {
      const { container } = render(PanelContent, { props: { type: 'chat', panelId: 'left' } });
      const content = container.querySelector('[data-testid="panel-content-chat"]');
      expect(content).toBeTruthy();
    });

    it('renders content area for review type', () => {
      const { container } = render(PanelContent, { props: { type: 'review', panelId: 'right' } });
      const content = container.querySelector('[data-testid="panel-content-review"]');
      expect(content).toBeTruthy();
    });

    it('renders content area for terminal type', () => {
      const { container } = render(PanelContent, { props: { type: 'terminal', panelId: 'left' } });
      const content = container.querySelector('[data-testid="panel-content-terminal"]');
      expect(content).toBeTruthy();
    });

    it('has no content area when type is null', () => {
      const { container } = render(PanelContent, { props: { type: null, panelId: 'left' } });
      const chatContent = container.querySelector('[data-testid="panel-content-chat"]');
      const reviewContent = container.querySelector('[data-testid="panel-content-review"]');
      const terminalContent = container.querySelector('[data-testid="panel-content-terminal"]');
      expect(chatContent).toBeNull();
      expect(reviewContent).toBeNull();
      expect(terminalContent).toBeNull();
    });
  });

  describe('structure', () => {
    it('fills available height', () => {
      const { container } = render(PanelContent, { props: { type: 'chat', panelId: 'left' } });
      const wrapper = container.querySelector('[data-panel-id="left"]');
      expect(wrapper?.classList.contains('h-full')).toBe(true);
    });
  });
});
