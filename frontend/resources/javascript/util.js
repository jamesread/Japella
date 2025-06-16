export	function waitForClient() {
  return new Promise((resolve) => {
    const interval = setInterval(() => {
      if (window.client) {
        clearInterval(interval)
        resolve()
      }
    }, 50)
  })
}
