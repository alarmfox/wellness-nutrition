import type { User } from '@prisma/client';
import { DateTime } from 'luxon';
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
    logo: env.NEXTAUTH_URL + '/logo_big.png'
  },
});

export async function sendWelcomeEmail({ email: userEmail, firstName }: User, verificationUrl: string) {
  const email: Mailgen.Content = {
    body: {
      greeting: 'Ciao',
      name: `${firstName}`,
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
      outro: `Hai bisogno di aiuto? Invia un messaggio a ${env.EMAIL_NOTIFY_ADDRESS} e saremo felici di aiutarti`
    }
  };

  await sendEmail(userEmail, env.EMAIL_FROM, 'Benvenuto in Wellness & Nutrition', email);
}

export async function sendResetEmail({ email: userEmail, firstName }: User, verificationUrl: string) {
  const email: Mailgen.Content = {
    body: {
      greeting: 'Ciao',
      name: `${firstName}`,
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

  await sendEmail(userEmail, env.EMAIL_FROM, 'Ripristino password', email);
}

export async function sendOnNewBooking(user: User, startsAt: Date) {
  const email: Mailgen.Content = {
    body: {
      greeting: 'Ciao',
      name: `amministratore`,
      signature: 'Grazie per averci scelto',
      intro: `Una nuova prenotazione è stata inserita da ${user.firstName} ${user.lastName} per 
      ${DateTime.fromJSDate(startsAt).toLocaleString(DateTime.DATETIME_MED_WITH_WEEKDAY, { locale: 'it' })}`,
      title: 'Nuova prenotazione',
      outro: 'Hai bisogno di aiuto? Rispondi a questa email e saremo felici di aiutarti'
    }
  };

  await sendEmail(env.EMAIL_NOTIFY_ADDRESS, env.EMAIL_FROM, 'Nuova prenotazione', email)

}

export async function sendOnDeleteBooking(user: User, startsAt: Date) {
  const email: Mailgen.Content = {
    body: {
      greeting: 'Ciao',
      name: `amministratore`,
      signature: 'Grazie per averci scelto',
      intro: `Una prenotazione è stata cancellata da ${user.firstName} ${user.lastName} per 
      ${DateTime.fromJSDate(startsAt).toLocaleString(DateTime.DATETIME_MED_WITH_WEEKDAY, { locale: 'it' })}`,
      title: 'Prenotazione cancellata',
      outro: 'Hai bisogno di aiuto? Rispondi a questa email e saremo felici di aiutarti'
    }
  };

  await sendEmail(env.EMAIL_NOTIFY_ADDRESS, env.EMAIL_FROM, 'Prenotazione cancellata', email)

}
async function sendEmail(to: string, from: string, subject: string, content: Mailgen.Content): Promise<void> {
  // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
  const html = mailGenerator.generate(content);

  const mailOptions = {
    from,
    to,
    subject,
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    html,
  };

  await new Promise((res, rej) => {
    transport.sendMail(mailOptions, (error, info) => {
      if (error) {
        console.error(error);
        rej(error)
      } else {
        console.log(info);
        res(info);
      }
    });
  })
}