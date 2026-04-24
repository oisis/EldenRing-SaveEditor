import {sfLog, sfLoading, sfDone} from '../components/ToastBar';

let idCounter = 0;

const toast = Object.assign(
    (msg: string) => { sfLog('info', msg); },
    {
        success: (msg: string, opts?: { id?: string }) => {
            if (opts?.id) sfDone(opts.id);
            sfLog('info', msg);
        },
        error: (msg: string, opts?: { id?: string }) => {
            if (opts?.id) sfDone(opts.id);
            sfLog('error', msg);
        },
        loading: (msg: string) => {
            const id = `sf-loading-${++idCounter}`;
            sfLoading(id, msg);
            return id;
        },
        dismiss: (id?: string) => {
            if (id) sfDone(id);
        },
    }
);

export default toast;
