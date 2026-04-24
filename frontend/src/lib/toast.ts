import {sfLog} from '../components/ToastBar';

let idCounter = 0;

const toast = Object.assign(
    (msg: string) => { sfLog('info', msg); },
    {
        success: (msg: string, _opts?: { id?: string }) => {
            sfLog('info', msg);
        },
        error: (msg: string, _opts?: { id?: string }) => {
            sfLog('error', msg);
        },
        loading: (msg: string) => {
            sfLog('info', msg);
            return `sf-loading-${++idCounter}`;
        },
        dismiss: (_id?: string) => {},
    }
);

export default toast;
