export function MyUtil(): number {
    return 222;
}

export async function DoPromise(): Promise<boolean> {
    return new Promise((res, rej) => {
        setTimeout(() => {
            res(true);
        }, 150);
    })
}