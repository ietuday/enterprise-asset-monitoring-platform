require("dotenv").config();

const express = require("express");
const cors = require("cors");
const helmet = require("helmet");
const morgan = require("morgan");

const { initDb } = require("./src/db/postgres");
const authRoutes = require("./src/routes/auth.routes");

const app = express();

const PORT = process.env.PORT || 4001;

app.use(helmet());
app.use(cors());
app.use(morgan("combined"));
app.use(express.json());

app.get("/health", (req, res) => {
  res.status(200).json({
    service: "auth-service",
    status: "healthy",
  });
});

app.use("/auth", authRoutes);

app.use((req, res) => {
  res.status(404).json({
    error: "route not found",
    path: req.originalUrl,
  });
});

async function startServer() {
  try {
    await initDb();

    app.listen(PORT, () => {
      console.log(`auth-service running on port ${PORT}`);
    });
  } catch (err) {
    console.error("failed to start auth-service:", err);
    process.exit(1);
  }
}

startServer();