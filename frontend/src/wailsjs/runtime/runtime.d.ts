export function EventsOn(eventName: string, callback: (...data: any[]) => void): void;
export function EventsOff(eventName: string, ...additionalEventNames: string[]): void;
export function EventsOnce(eventName: string, callback: (...data: any[]) => void): void;
