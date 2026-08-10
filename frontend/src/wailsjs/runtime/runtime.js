export function EventsOn(eventName, callback) {
    if (window['runtime'] && window['runtime']['EventsOn']) {
        return window['runtime']['EventsOn'](eventName, callback);
    }
}
export function EventsOff(eventName, ...additionalEventNames) {
    if (window['runtime'] && window['runtime']['EventsOff']) {
        return window['runtime']['EventsOff'](eventName, ...additionalEventNames);
    }
}
export function EventsOnce(eventName, callback) {
    if (window['runtime'] && window['runtime']['EventsOnce']) {
        return window['runtime']['EventsOnce'](eventName, callback);
    }
}
