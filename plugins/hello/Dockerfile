FROM node:20-alpine
WORKDIR /app
COPY dist/ ./dist/
RUN npm install -g serve@14
EXPOSE 3001
CMD ["serve", "-s", "dist", "-l", "3001", "--no-clipboard"]
