export	function waitForClient() {
  return new Promise((resolve, reject) => {
    const TIMEOUT_MS = 10_000

    const interval = setInterval(() => {
      if (window.client) {
        clearInterval(interval)
        clearTimeout(timeout)
        resolve(window.client)
      }
    }, 50)

    const timeout = setTimeout(() => {
      clearInterval(interval)
      reject(new Error('Client not found within timeout period'))
    }, TIMEOUT_MS)
  })
}
