export function MyUtil(): number {
    return 321;
}

export async function DoPromise(timeout): Promise<boolean> {
    return new Promise((res, rej) => {
        setTimeout(() => {
            res(true);
        }, timeout);
    })
}