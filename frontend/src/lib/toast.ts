import {sfLog} from '../components/ToastBar';
import hotToast from 'react-hot-toast';

const toast = Object.assign(
    (msg: string) => { sfLog('info', msg); },
    {
        success: (msg: string, opts?: { id?: string }) => {
            sfLog('info', msg);
            if (opts?.id) hotToast.dismiss(opts.id);
        },
        error: (msg: string, opts?: { id?: string }) => {
            sfLog('error', msg);
            if (opts?.id) hotToast.dismiss(opts.id);
        },
        loading: (msg: string) => {
            sfLog('info', msg);
            return hotToast.loading(msg, {
                style: { background: 'var(--color-card)', color: 'var(--color-foreground)', border: '1px solid var(--color-border)' },
            });
        },
        dismiss: hotToast.dismiss,
    }
);

export default toast;
