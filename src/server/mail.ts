import type { User } from '@prisma/client';
import Mailgen from 'mailgen';
import * as nodemailer from 'nodemailer';
import { env } from '../env/server.mjs';

// Nodemailer configuration
// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
const transport = nodemailer.createTransport({
  port: +env.EMAIL_SERVER_PORT,
  host: env.EMAIL_SERVER_HOST,
  secure: +env.EMAIL_SERVER_PORT === 465,
  auth: {
    user: env.EMAIL_SERVER_USER,
    pass: env.EMAIL_SERVER_PASSWORD
  }
});
const mailGenerator = new Mailgen({
  theme: 'default',
  product: {
    // Appears in header & footer of e-mails
    name: 'Wellness & Nutrition',
    link: env.NEXTAUTH_URL,
    copyright: 'Tutti i diritti riservati',
    // Optional product logo
    logo: env.NEXTAUTH_URL + '/logo.jpeg'
  },
});

export function sendWelcomeEmail(user: User, verificationUrl: string) {
  const email: Mailgen.Content = {
    body: {
      greeting: 'Ciao',
      name: `${user.firstName}`,
      signature: 'Grazie per averci scelto',
      intro: 'Benvenuto in Wellness & Nutrition.',
      action: {
        instructions: 'Per verificare il tuo account e impostare una password, clicca il pulsante di seguito:',
        button: {
          color: '#22BC66', // Optional action button color
          text: 'Conferma account',
          link: verificationUrl,
        }
      },
      outro: 'Hai bisogno di aiuto? Rispondi a questa email e saremo felici di aiutarti'
    }
  };

  // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
  const html = mailGenerator.generate(email);

  const mailOptions = {
    from: env.EMAIL_FROM,
    to: user.email,
    subject: 'Benvenuto in Wellness e Nutrition',
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    html,
  };

  transport.sendMail(mailOptions, (error) => {
    if (error) throw error;
  });
}

export function sendResetEmail(user: User, verificationUrl: string) {
  const email: Mailgen.Content = {
    body: {
      greeting: 'Ciao',
      name: `${user.firstName}`,
      signature: 'Grazie per averci scelto',
      intro: 'Ricevi questa email per ripristinare la credenziali.',
      action: {
        instructions: 'Per ripristinare le credenziali, clicca il pulsante di seguito:',
        button: {
          color: '#22BC66', // Optional action button color
          text: 'Ripristina credenziali',
          link: verificationUrl,
        }
      },
      outro: 'Hai bisogno di aiuto? Rispondi a questa email e saremo felici di aiutarti'
    }
  };

  // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
  const html = mailGenerator.generate(email);

  const mailOptions = {
    from: env.EMAIL_FROM,
    to: user.email,
    subject: 'Ripristino password',
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    html,
  };

  transport.sendMail(mailOptions, (error) => {
    if (error) throw error;
  });
}

