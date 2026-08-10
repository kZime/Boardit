// frontend/orval.config.cjs
// This file is used to generate the API client code from backend/docs/api/api-contract-v1.yaml


module.exports = {
  boardit: {
    input: '../backend/docs/api/api-contract-v1.yaml',
    output: {
      target: 'src/api/gen/client.ts',
      client: 'react-query',
      httpClient: 'axios',
      mock: true,
      baseUrl: '',
      schemas: 'src/api/gen/models',
      prettier: true,
      override: {
        mutator: { path: 'src/api/orval-axios.ts', name: 'orvalRequester' },
      },
    },
  },
}
