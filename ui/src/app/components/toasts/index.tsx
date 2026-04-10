import toast, { useToaster } from 'react-hot-toast/headless';
import { ToastNotification } from '@/app/components/carbon/toast';

export const Toast = () => {
  const { toasts, handlers } = useToaster();
  const { startPause, endPause, updateHeight } = handlers;

  return (
    <div
      onMouseEnter={startPause}
      onMouseLeave={endPause}
      className="absolute top-0 right-0 z-10 flex flex-col items-end"
    >
      {toasts.map(t => {
        const ref = (el: HTMLDivElement | null) => {
          if (el && typeof t.height !== 'number') {
            const height = el.getBoundingClientRect().height;
            updateHeight(t.id, height);
          }
        };

        const kind =
          t.type === 'success'
            ? 'success'
            : t.type === 'error'
              ? 'error'
              : 'info';

        return (
          <div ref={ref} key={t.id}>
            <ToastNotification
              kind={kind}
              title={t.message?.toString() ?? ''}
              onCloseButtonClick={() => toast.remove(t.id)}
            />
          </div>
        );
      })}
    </div>
  );
};
