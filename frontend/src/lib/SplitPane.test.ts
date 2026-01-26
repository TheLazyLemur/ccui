import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';
import SplitPane from './SplitPane.svelte';
import SplitPaneTestWrapper from './SplitPaneTestWrapper.svelte';

describe('SplitPane', () => {
  afterEach(() => {
    cleanup();
  });

  describe('with both panels', () => {
    it('renders with default 50% left size', () => {
      const { container } = render(SplitPaneTestWrapper);
      const leftPanel = container.querySelector('[data-testid="left-panel"]');
      expect(leftPanel).toBeTruthy();
      expect((leftPanel as HTMLElement).style.width).toBe('50%');
    });

    it('renders with custom leftSize prop', () => {
      const { container } = render(SplitPaneTestWrapper, { props: { leftSize: 30 } });
      const leftPanel = container.querySelector('[data-testid="left-panel"]');
      expect((leftPanel as HTMLElement).style.width).toBe('30%');
    });

    it('renders drag handle', () => {
      const { container } = render(SplitPaneTestWrapper);
      const handle = container.querySelector('[data-testid="drag-handle"]');
      expect(handle).toBeTruthy();
    });

    it('constrains size to minLeft', () => {
      const { container } = render(SplitPaneTestWrapper, { props: { leftSize: 10, minLeft: 20 } });
      const leftPanel = container.querySelector('[data-testid="left-panel"]');
      expect((leftPanel as HTMLElement).style.width).toBe('20%');
    });

    it('constrains size to minRight', () => {
      const { container } = render(SplitPaneTestWrapper, { props: { leftSize: 90, minRight: 20 } });
      const leftPanel = container.querySelector('[data-testid="left-panel"]');
      expect((leftPanel as HTMLElement).style.width).toBe('80%');
    });

    it('has handle element with correct class', () => {
      const { container } = render(SplitPaneTestWrapper);
      const handle = container.querySelector('[data-testid="drag-handle"]');
      expect(handle?.classList.contains('handle')).toBe(true);
    });

    it('renders both left and right panels', () => {
      const { container } = render(SplitPaneTestWrapper);
      const leftPanel = container.querySelector('[data-testid="left-panel"]');
      const rightPanel = container.querySelector('[data-testid="right-panel"]');
      expect(leftPanel).toBeTruthy();
      expect(rightPanel).toBeTruthy();
    });
  });

  describe('drag behavior', () => {
    it('adds dragging class on mousedown', async () => {
      const { container } = render(SplitPaneTestWrapper);
      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;

      await fireEvent.mouseDown(handle);
      await tick();

      expect(handle.classList.contains('dragging')).toBe(true);
    });

    it('removes dragging class on mouseup', async () => {
      const { container } = render(SplitPaneTestWrapper);
      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;

      await fireEvent.mouseDown(handle);
      await tick();
      expect(handle.classList.contains('dragging')).toBe(true);

      window.dispatchEvent(new MouseEvent('mouseup'));
      await tick();

      expect(handle.classList.contains('dragging')).toBe(false);
    });

    it('updates size on mousemove during drag', async () => {
      const { container } = render(SplitPaneTestWrapper, { props: { leftSize: 50, minLeft: 20, minRight: 20 } });
      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;
      const splitPane = container.querySelector('[data-testid="split-pane"]') as HTMLElement;
      const leftPanel = container.querySelector('[data-testid="left-panel"]') as HTMLElement;

      vi.spyOn(splitPane, 'getBoundingClientRect').mockReturnValue({
        left: 0, right: 1000, width: 1000,
        top: 0, bottom: 500, height: 500,
        x: 0, y: 0, toJSON: () => ({})
      });

      await fireEvent.mouseDown(handle);
      await tick();

      window.dispatchEvent(new MouseEvent('mousemove', { clientX: 300 }));
      await tick();

      expect(leftPanel.style.width).toBe('30%');
    });

    it('respects minLeft constraint during drag', async () => {
      const { container } = render(SplitPaneTestWrapper, { props: { leftSize: 50, minLeft: 20, minRight: 20 } });
      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;
      const splitPane = container.querySelector('[data-testid="split-pane"]') as HTMLElement;
      const leftPanel = container.querySelector('[data-testid="left-panel"]') as HTMLElement;

      vi.spyOn(splitPane, 'getBoundingClientRect').mockReturnValue({
        left: 0, right: 1000, width: 1000,
        top: 0, bottom: 500, height: 500,
        x: 0, y: 0, toJSON: () => ({})
      });

      await fireEvent.mouseDown(handle);
      await tick();

      window.dispatchEvent(new MouseEvent('mousemove', { clientX: 50 }));
      await tick();

      expect(leftPanel.style.width).toBe('20%');
    });

    it('respects minRight constraint during drag', async () => {
      const { container } = render(SplitPaneTestWrapper, { props: { leftSize: 50, minLeft: 20, minRight: 20 } });
      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;
      const splitPane = container.querySelector('[data-testid="split-pane"]') as HTMLElement;
      const leftPanel = container.querySelector('[data-testid="left-panel"]') as HTMLElement;

      vi.spyOn(splitPane, 'getBoundingClientRect').mockReturnValue({
        left: 0, right: 1000, width: 1000,
        top: 0, bottom: 500, height: 500,
        x: 0, y: 0, toJSON: () => ({})
      });

      await fireEvent.mouseDown(handle);
      await tick();

      window.dispatchEvent(new MouseEvent('mousemove', { clientX: 950 }));
      await tick();

      expect(leftPanel.style.width).toBe('80%');
    });

    it('emits resize event with final size on drag end', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(SplitPaneTestWrapper, { props: { leftSize: 50 } });
      component.$on('resize', mockHandler);

      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;
      const splitPane = container.querySelector('[data-testid="split-pane"]') as HTMLElement;

      vi.spyOn(splitPane, 'getBoundingClientRect').mockReturnValue({
        left: 0, right: 1000, width: 1000,
        top: 0, bottom: 500, height: 500,
        x: 0, y: 0, toJSON: () => ({})
      });

      await fireEvent.mouseDown(handle);
      await tick();

      window.dispatchEvent(new MouseEvent('mousemove', { clientX: 400 }));
      await tick();

      window.dispatchEvent(new MouseEvent('mouseup'));
      await tick();

      expect(mockHandler).toHaveBeenCalledTimes(1);
      expect(mockHandler.mock.calls[0][0].detail).toBe(40);
    });

    it('does not emit resize if not dragging', async () => {
      const mockHandler = vi.fn();
      const { component } = render(SplitPaneTestWrapper);
      component.$on('resize', mockHandler);

      window.dispatchEvent(new MouseEvent('mouseup'));
      await tick();

      expect(mockHandler).not.toHaveBeenCalled();
    });
  });

  describe('keyboard navigation', () => {
    it('decreases size on ArrowLeft', async () => {
      const { container } = render(SplitPaneTestWrapper, { props: { leftSize: 50 } });
      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;
      const leftPanel = container.querySelector('[data-testid="left-panel"]') as HTMLElement;

      await fireEvent.keyDown(handle, { key: 'ArrowLeft' });
      await tick();

      expect(leftPanel.style.width).toBe('45%');
    });

    it('increases size on ArrowRight', async () => {
      const { container } = render(SplitPaneTestWrapper, { props: { leftSize: 50 } });
      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;
      const leftPanel = container.querySelector('[data-testid="left-panel"]') as HTMLElement;

      await fireEvent.keyDown(handle, { key: 'ArrowRight' });
      await tick();

      expect(leftPanel.style.width).toBe('55%');
    });

    it('respects minLeft constraint on keyboard resize', async () => {
      const { container } = render(SplitPaneTestWrapper, { props: { leftSize: 22, minLeft: 20 } });
      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;
      const leftPanel = container.querySelector('[data-testid="left-panel"]') as HTMLElement;

      await fireEvent.keyDown(handle, { key: 'ArrowLeft' });
      await tick();

      expect(leftPanel.style.width).toBe('20%');
    });

    it('respects minRight constraint on keyboard resize', async () => {
      const { container } = render(SplitPaneTestWrapper, { props: { leftSize: 78, minRight: 20 } });
      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;
      const leftPanel = container.querySelector('[data-testid="left-panel"]') as HTMLElement;

      await fireEvent.keyDown(handle, { key: 'ArrowRight' });
      await tick();

      expect(leftPanel.style.width).toBe('80%');
    });

    it('emits resize event on keyboard resize', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(SplitPaneTestWrapper, { props: { leftSize: 50 } });
      component.$on('resize', mockHandler);

      const handle = container.querySelector('[data-testid="drag-handle"]') as HTMLElement;

      await fireEvent.keyDown(handle, { key: 'ArrowLeft' });
      await tick();

      expect(mockHandler).toHaveBeenCalledTimes(1);
      expect(mockHandler.mock.calls[0][0].detail).toBe(45);
    });
  });

  describe('single-panel mode', () => {
    it('takes full width when no right slot content', () => {
      const { container } = render(SplitPane);
      const leftPanel = container.querySelector('[data-testid="left-panel"]');
      expect((leftPanel as HTMLElement).style.width).toBe('100%');
    });

    it('hides drag handle when no right slot content', () => {
      const { container } = render(SplitPane);
      const handle = container.querySelector('[data-testid="drag-handle"]');
      expect(handle).toBeNull();
    });
  });
});
