export default class Notification {
  constructor(cssClass, title, message, url) {
    this.cssClass = cssClass || 'info';
    this.title = title;
    this.message = message;
    this.url = url;

    this.notificationList = document.getElementById('notification-list');

    if (!this.notificationList) {
      this.notificationList = document.createElement('div');
      this.notificationList.setAttribute('id', 'notification-list');
    }

    document.body.appendChild(this.notificationList);
  }

  show() {
    const notification = document.createElement('div');
    notification.className = `notification ${this.cssClass}`;

    const titleElement = document.createElement('strong');
    titleElement.textContent = this.title;

    const messageElement = document.createElement('span');
    messageElement.textContent = this.message;

    let validUrl = false;

    try {
      validUrl = new URL(this.url).toString()
    } catch (e) { }

    if (validUrl) {
      const linkElement = document.createElement('a');
      linkElement.href = validUrl;
      linkElement.innerText = 'Link';
      linkElement.target = '_blank';
      messageElement.appendChild(linkElement);
    }

    notification.appendChild(titleElement);
    notification.appendChild(messageElement);

    this.notificationList.appendChild(notification);

    setTimeout(() => {
      notification.remove();
    }, 5000); // Remove after 5 seconds

    notification.onclick = () => {
      notification.remove();
    }
  }
}
