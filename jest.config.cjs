module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/jest.setup.ts'],
  moduleNameMapper: { '^@/(.*)$': '<rootDir>/src/$1', '\\.(css|less|scss|sass)$': 'identity-obj-proxy' },
  transform: { '^.+\\.tsx?$': ['ts-jest', { tsconfig: 'tsconfig.app.json' }] }
};
