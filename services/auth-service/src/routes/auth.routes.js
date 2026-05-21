const express = require("express");
const bcrypt = require("bcryptjs");
const jwt = require("jsonwebtoken");

const { pool } = require("../db/postgres");
const { authenticate } = require("../middleware/auth.middleware");

const router = express.Router();

function createToken(user) {
  return jwt.sign(
    {
      id: user.id,
      email: user.email,
      role: user.role,
      name: user.name,
    },
    process.env.JWT_SECRET || "supersecretkey",
    {
      expiresIn: "1h",
    }
  );
}

router.post("/register", async (req, res) => {
  try {
    const { name, email, password, role } = req.body;

    if (!name || !email || !password) {
      return res.status(400).json({
        error: "name, email and password are required",
      });
    }

    const passwordHash = await bcrypt.hash(password, 10);

    const result = await pool.query(
      `
      INSERT INTO users (name, email, password_hash, role)
      VALUES ($1, $2, $3, $4)
      RETURNING id, name, email, role, created_at;
      `,
      [name, email, passwordHash, role || "VIEWER"]
    );

    const user = result.rows[0];
    const token = createToken(user);

    return res.status(201).json({
      user,
      token,
    });
  } catch (err) {
    if (err.code === "23505") {
      return res.status(409).json({
        error: "email already exists",
      });
    }

    console.error("register failed:", err);
    return res.status(500).json({
      error: "internal server error",
    });
  }
});

router.post("/login", async (req, res) => {
  try {
    const { email, password } = req.body;

    if (!email || !password) {
      return res.status(400).json({
        error: "email and password are required",
      });
    }

    const result = await pool.query(
      `
      SELECT id, name, email, password_hash, role, created_at
      FROM users
      WHERE email = $1;
      `,
      [email]
    );

    if (result.rowCount === 0) {
      return res.status(401).json({
        error: "invalid email or password",
      });
    }

    const userRecord = result.rows[0];

    const isPasswordValid = await bcrypt.compare(
      password,
      userRecord.password_hash
    );

    if (!isPasswordValid) {
      return res.status(401).json({
        error: "invalid email or password",
      });
    }

    const user = {
      id: userRecord.id,
      name: userRecord.name,
      email: userRecord.email,
      role: userRecord.role,
      created_at: userRecord.created_at,
    };

    const token = createToken(user);

    return res.status(200).json({
      user,
      token,
    });
  } catch (err) {
    console.error("login failed:", err);
    return res.status(500).json({
      error: "internal server error",
    });
  }
});

router.get("/me", authenticate, async (req, res) => {
  return res.status(200).json({
    user: req.user,
  });
});

module.exports = router;