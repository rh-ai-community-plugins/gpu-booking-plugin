const { merge } = require('webpack-merge');
const common = require('./webpack.common.js');
const path = require('path');

module.exports = merge(common, {
  mode: 'development',
  devtool: 'eval-source-map',
  devServer: {
    port: parseInt(process.env.PORT, 10) || 9500, // [PLUGIN-SPECIFIC] dev port
    historyApiFallback: true,
    hot: true,
    headers: {
      // remoteEntry CORS headers so the dashboard host can load the plugin
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, PATCH, OPTIONS',
      'Access-Control-Allow-Headers': 'X-Requested-With, Content-Type, Authorization',
    },
    proxy: [
      {
        context: ['/gpu-booking/api'], // [PLUGIN-SPECIFIC] Go backend proxy — must come before the general proxy
        target: 'http://localhost:3000',
        pathRewrite: { '^/gpu-booking/api': '/api' },
      },
      {
        context: ['/gpu-booking'], // [PLUGIN-SPECIFIC] must match route prefix
        target: 'http://localhost:8443',
        pathRewrite: { '^/gpu-booking': '/gpu-booking' },
      },
    ],
  },
  optimization: {
    runtimeChunk: false,
    splitChunks: false,
  },
});
