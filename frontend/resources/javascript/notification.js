export default class Notification {
  constructor(cssClass, title, message) {
    this.cssClass = cssClass || 'info';
    this.title = title;
    this.message = message;
  }

  show() {
    const notification = document.createElement('div');
    notification.className = `notification ${this.cssClass}`;

    const titleElement = document.createElement('strong');
    titleElement.textContent = this.title;

    const messageElement = document.createElement('span');
    messageElement.textContent = this.message;

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
