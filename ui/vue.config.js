
console.log("** loading vue.config.js ********************")

module.exports = {
    devServer: {
      disableHostCheck: true,
      proxy: {
        '/api/image': {
          target: 'http://localhost:8000'
        },
        '/api': {
          target: 'http://localhost:4444',
          pathRewrite: {
            '/api': '/',
          },
        }
      }
    }
  }