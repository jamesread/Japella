export default class Notification {
  constructor(cssClass, title, message, url) {
    this.cssClass = cssClass || 'info';
    this.title = title;
    this.message = message;
    this.url = url;
  }

  show() {
    const notification = document.createElement('div');
    notification.className = `notification ${this.cssClass}`;

    const titleElement = document.createElement('strong');
    titleElement.textContent = this.title;

    const messageElement = document.createElement('span');
    messageElement.textContent = this.message;

    if (this.url) {
      const linkElement = document.createElement('a');
      linkElement.href = this.url;
      linkElement.innerText = 'Link';
      linkElement.target = '_blank';
      messageElement.appendChild(linkElement);
    }

    notification.appendChild(titleElement);
    notification.appendChild(messageElement);

    document.body.appendChild(notification);

    setTimeout(() => {
      notification.remove();
    }, 5000); // Remove after 5 seconds

    notification.onclick = () => {
      notification.remove();
    }
  }
}
