export function MyUtil(): number {
    return 331;
}

export async function DoPromise(): Promise<boolean> {
    return new Promise((res, rej) => {
        setTimeout(() => {
            res(true);
        }, 150);
    })
}