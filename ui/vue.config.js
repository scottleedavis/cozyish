
console.log("** loading vue.config.js ********************")

const apiTarget = 'http://localhost:8000';
const crawlerTarget = 'http://localhost:4444';
module.exports = {
    devServer: {
      disableHostCheck: true,
      proxy: {
        '/api/image': {
          target: apiTarget,
        },
        '/api': {
          target: crawlerTarget,
          pathRewrite: {
            '/api': '/',
          },
        }
      }
    }
  }