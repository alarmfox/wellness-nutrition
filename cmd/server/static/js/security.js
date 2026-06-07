function getCookie(name) {
    const cookie = document.cookie
        .split(';')
        .map(c => c.trim())
        .find(c => c.startsWith(name + '='));
    if (!cookie) return '';

    return decodeURIComponent(cookie.substring(name.length + 1));
}
