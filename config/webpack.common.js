const HtmlWebpackPlugin = require('html-webpack-plugin');
const { ModuleFederationPlugin } = require('webpack').container;
const path = require('path');
const { name, version, 'module-federation': moduleFederation } = require('../package.json');
const stylePaths = require('./stylePaths');

const remoteEntry = path.posix.join(moduleFederation.remoteEntry);

module.exports = {
  entry: './src/index.ts',
  output: {
    publicPath: 'auto',
    filename: '[name].[contenthash].js',
  },
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        use: {
          loader: 'ts-loader',
          options: {
            transpileOnly: true,
            compilerOptions: {
              noEmit: false,
            },
          },
        },
        include: /src/,
        exclude: /\.test\.(ts|tsx)$/,
      },
      {
        test: /\.css$/,
        use: [
          {
            loader: 'style-loader',
          },
          {
            loader: 'css-loader',
          },
        ],
      },
      {
        // [PLUGIN-SPECIFIC] Help pages import markdown docs as strings
        test: /\.md$/,
        type: 'asset/source',
      },
      {
        test: /\.(png|jpg|jpeg|gif|svg|woff2?|eot|ttf|otf)$/i,
        type: 'asset/resource',
      },
      {
        // ESM-only deps (react-markdown, remark-gfm, rehype-raw) may use
        // extensionless imports — disable full-specifier resolution for JS
        test: /\.(m?js)$/,
        resolve: {
          fullySpecified: false,
        },
      },
    ],
  },
  resolve: {
    extensions: ['.js', '.ts', '.tsx', '.jsx'],
    alias: {
      '~': path.resolve(__dirname, '../src'),
    },
  },
  plugins: [
    new HtmlWebpackPlugin({
      template: path.resolve(__dirname, '../src/index.html'),
    }),
    new ModuleFederationPlugin({
      name: 'gpuBooking', // [PLUGIN-SPECIFIC] must match package.json and plugin.yaml
      filename: remoteEntry,
      exposes: {
        './extensions': './src/rhoai/extensions.ts',
        './Icon': './src/app/components/GpuBookingNavIcon.tsx',
      },
      shared: {
        react: {
          singleton: true,
          requiredVersion: '^18',
        },
        'react-dom': {
          singleton: true,
          requiredVersion: '^18',
        },
        'react-router-dom': {
          singleton: true,
          requiredVersion: '^7',
        },
        '@patternfly/react-core': {
          singleton: true,
          requiredVersion: '^6',
        },
        '@openshift/dynamic-plugin-sdk': {
          singleton: true,
          requiredVersion: '^5',
        },
      },
    }),
  ],
};
