import * as nodemailer from 'nodemailer';
import { env } from '../env/server.mjs';

// Nodemailer configuration
// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
const transporter = nodemailer.createTransport({
    port: +env.EMAIL_SERVER_PORT,
    host: env.EMAIL_SERVER_HOST,
    secure: +env.EMAIL_SERVER_PORT === 465,
    auth: {
        user: env.EMAIL_SERVER_USER,
        pass: env.EMAIL_SERVER_PASSWORD
    }
});

export function sendVerificationEmail(to: string, verificationUrl: string) {
    console.log(to, verificationUrl);
}